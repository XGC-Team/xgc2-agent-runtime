package agentruntime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

func NewDriver(p Profile, sink Sink, ask Ask) (Driver, error) {
	if _, err := commandArgs(p.Provider); err != nil {
		return nil, err
	}
	if p.Provider == "claude" {
		return &claudeDriver{profile: p, sink: sink, ask: ask}, nil
	}
	return &rpcDriver{profile: p, sink: sink, ask: ask}, nil
}

type nativeTerminal struct {
	id      string
	status  string
	message string
}

type rpcDriver struct {
	profile          Profile
	sink             Sink
	ask              Ask
	peer             *rpcPeer
	process          *child
	mu               sync.Mutex
	nativeSession    string
	nativeTurn       string
	turn             string
	turnDone         chan nativeTerminal
	messageSerial    uint64
	messageID        string
	messageRole      string
	turnContext      context.Context
	acpSetup         map[string]any
	cwd              string
	launchPermission string
}

func (d *rpcDriver) emit(e Event) {
	d.mu.Lock()
	if e.TurnID == "" {
		e.TurnID = d.turn
	}
	if d.profile.Provider == "codex" {
		e.AgentThreadID = d.nativeSession
		e.AgentTurnID = d.nativeTurn
	}
	d.mu.Unlock()
	if err := d.sink(e); err != nil {
		go d.Close()
	}
}
func (d *rpcDriver) Open(ctx context.Context, cwd, nativeID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	args, _ := commandArgs(d.profile.Provider)
	if d.profile.Provider == "cursor" {
		var err error
		args, err = cursorPermissionArgs(d.profile.Defaults.Permission)
		if err != nil {
			return err
		}
	}
	if d.profile.Provider == "grok" {
		var err error
		args, err = grokPermissionArgs(d.profile.Defaults.Permission)
		if err != nil {
			return err
		}
	}
	args, environment, err := localMCPLaunch(ctx, d.profile.Provider, args)
	if err != nil {
		return err
	}
	c, err := startChild(d.profile, args, cwd, environment...)
	if err != nil {
		return err
	}
	d.mu.Lock()
	d.process = c
	d.cwd = cwd
	d.launchPermission = d.profile.Defaults.Permission
	d.mu.Unlock()
	peer := newPeer(c.stdin, c.stdout, d.profile.Provider != "codex", d.onNotification, d.onRequest)
	d.mu.Lock()
	d.peer = peer
	d.mu.Unlock()
	stopOpen := context.AfterFunc(ctx, func() { _ = d.Close() })
	defer stopOpen()
	go func() { <-peer.done; _ = c.Wait() }()
	handshake, cancel := context.WithTimeout(ctx, AgentSettingsTimeout)
	defer cancel()
	var result map[string]any
	if d.profile.Provider == "codex" {
		_, err = d.peer.Call(handshake, "initialize", map[string]any{"clientInfo": map[string]any{"name": "xgc-native-agent", "title": "XGC Native Agent", "version": "0.1.0"}, "capabilities": map[string]any{"experimentalApi": false}})
		if err == nil {
			err = d.peer.Notify("initialized", map[string]any{})
		}
		if err != nil {
			return err
		}
		account, e := d.peer.Call(handshake, "account/read", map[string]any{"refreshToken": false})
		if e != nil {
			return e
		}
		if text(obj(account["account"]), "type") != "chatgpt" {
			return errors.New("native ChatGPT login required; Research OS will not fall back to API billing")
		}
		params := map[string]any{"cwd": cwd}
		localMCPThreadContext(ctx, params)
		if err = codexThreadOptions(params, d.profile.Defaults); err != nil {
			return err
		}
		method := "thread/start"
		if nativeID != "" {
			method = "thread/resume"
			params["threadId"] = nativeID
			// Only the session metadata is consumed; excluding turns keeps resume cheap
			// and tolerant of historical item schema drift (t3code CodexSessionRuntime).
			params["excludeTurns"] = true
		}
		result, err = d.peer.Call(handshake, method, params)
		if err != nil && method == "thread/resume" && strings.Contains(err.Error(), "excludeTurns") {
			// Older pinned CLIs without excludeTurns: retry with full history.
			delete(params, "excludeTurns")
			result, err = d.peer.Call(handshake, method, params)
		}
		if err == nil {
			nativeID = text(obj(result["thread"]), "id")
		}
	} else {
		capabilities := map[string]any{"fs": map[string]any{"readTextFile": false, "writeTextFile": false}, "terminal": false}
		if d.profile.Provider == "cursor" {
			capabilities["_meta"] = map[string]any{"parameterizedModelPicker": true}
		}
		init, e := d.peer.Call(handshake, "initialize", map[string]any{"protocolVersion": 1, "clientInfo": map[string]any{"name": "xgc-native-agent", "version": "0.1.0"}, "clientCapabilities": capabilities})
		if e != nil {
			return e
		}
		if version, ok := init["protocolVersion"].(float64); !ok || version != 1 {
			return errors.New("unsupported ACP protocol version")
		}
		if d.profile.Provider == "cursor" {
			if _, err = d.peer.Call(handshake, "authenticate", map[string]any{"methodId": "cursor_login"}); err != nil {
				return errors.New("existing native Cursor login is unavailable")
			}
		}
		if d.profile.Provider == "grok" {
			cached := false
			for _, v := range arr(init["authMethods"]) {
				if text(obj(v), "id") == "cached_token" {
					cached = true
				}
			}
			if !cached {
				return errors.New("Grok cached native login is unavailable; log in outside Research OS")
			}
			if _, err = d.peer.Call(handshake, "authenticate", map[string]any{"methodId": "cached_token", "_meta": map[string]any{"headless": true}}); err != nil {
				return err
			}
		}
		servers, setupErr := localMCPACPServers(ctx, init)
		if setupErr != nil {
			return setupErr
		}
		params := map[string]any{"cwd": cwd, "mcpServers": servers}
		method := "session/new"
		if nativeID != "" {
			load, _ := obj(init["agentCapabilities"])["loadSession"].(bool)
			if !load {
				return errors.New("this native ACP version cannot resume a session")
			}
			method = "session/load"
			params["sessionId"] = nativeID
		}
		result, err = d.peer.Call(handshake, method, params)
		if err == nil && nativeID == "" {
			nativeID = text(result, "sessionId")
		}
	}
	if err != nil {
		return err
	}
	if nativeID == "" || len(nativeID) > 512 {
		return errors.New("native session identity missing")
	}
	d.mu.Lock()
	d.nativeSession = nativeID
	d.acpSetup = result
	d.mu.Unlock()
	if d.profile.Provider != "codex" {
		if err = d.applyACPOptions(ctx, d.profile.Defaults); err != nil {
			return err
		}
	}
	d.emit(Event{Kind: "session.identity", AgentSessionID: nativeID})
	return nil
}
func (d *rpcDriver) Prompt(ctx context.Context, turn, prompt string) error {
	return d.PromptWithOptions(ctx, turn, prompt, d.profile.Defaults)
}
func (d *rpcDriver) PromptWithOptions(ctx context.Context, turn, prompt string, options AgentOptions) error {
	if d.profile.Provider == "grok" || d.profile.Provider == "cursor" {
		d.mu.Lock()
		changed := acpLaunchPermission(d.profile.Provider, options.Permission) != acpLaunchPermission(d.profile.Provider, d.launchPermission)
		cwd, native := d.cwd, d.nativeSession
		d.mu.Unlock()
		if changed {
			if err := d.Close(); err != nil {
				return err
			}
			d.profile.Defaults = options
			if err := d.Open(ctx, cwd, native); err != nil {
				return err
			}
		}
	}
	d.mu.Lock()
	d.turn = turn
	d.nativeTurn = ""
	d.turnDone = make(chan nativeTerminal, 8)
	d.turnContext = ctx
	d.messageSerial = 0
	d.messageID = ""
	d.messageRole = ""
	native := d.nativeSession
	done := d.turnDone
	d.mu.Unlock()
	defer func() { d.mu.Lock(); d.turnContext = nil; d.turn = ""; d.nativeTurn = ""; d.mu.Unlock() }()
	if d.profile.Provider != "codex" {
		if err := d.applyACPOptions(ctx, options); err != nil {
			return err
		}
		result, err := d.peer.Call(ctx, "session/prompt", map[string]any{"sessionId": native, "prompt": localMCPACPContent(ctx, prompt)})
		if err != nil {
			return err
		}
		status := "incomplete"
		switch text(result, "stopReason") {
		case "end_turn":
			status = "completed"
		case "cancelled":
			status = "cancelled"
		case "refusal":
			status = "refused"
		}
		d.emit(Event{Kind: "turn.end", Status: status, SourceMethod: "session/prompt"})
		return nil
	}
	params := map[string]any{"threadId": native, "input": []any{map[string]any{"type": "text", "text": prompt}}}
	if err := codexTurnOptions(params, options); err != nil {
		return err
	}
	result, err := d.peer.Call(ctx, "turn/start", params)
	if err != nil {
		return err
	}
	nativeTurn := text(obj(result["turn"]), "id")
	if nativeTurn == "" {
		return errors.New("native turn identity missing")
	}
	d.mu.Lock()
	if d.nativeTurn == "" {
		d.nativeTurn = nativeTurn
	}
	same := d.nativeTurn == nativeTurn
	d.mu.Unlock()
	if !same {
		return errors.New("native turn identity mismatch")
	}
	for {
		select {
		case terminal := <-done:
			if terminal.id != nativeTurn {
				continue
			}
			d.emit(Event{Kind: "turn.end", Status: terminal.status, Text: terminal.message, SourceMethod: "turn/completed"})
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-d.peer.done:
			return io.EOF
		}
	}
}
func (d *rpcDriver) Cancel(ctx context.Context) error {
	d.mu.Lock()
	native, turn := d.nativeSession, d.nativeTurn
	d.mu.Unlock()
	if d.peer == nil {
		return ErrUnavailable
	}
	if d.profile.Provider == "codex" {
		if turn == "" {
			return errors.New("native turn not yet acknowledged; close session to stop the worker")
		}
		_, err := d.peer.Call(ctx, "turn/interrupt", map[string]any{"threadId": native, "turnId": turn})
		return err
	}
	return d.peer.Notify("session/cancel", map[string]any{"sessionId": native})
}
func (d *rpcDriver) Close() error {
	d.mu.Lock()
	peer, process := d.peer, d.process
	d.mu.Unlock()
	if peer != nil {
		_ = peer.Close()
	}
	if process != nil {
		process.Stop()
	}
	return nil
}
func (d *rpcDriver) matches(params map[string]any) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := text(params, "sessionId")
	if d.profile.Provider == "codex" {
		id = text(params, "threadId")
	}
	// Notifications emitted during session/load are not replayed into a new local
	// turn. Display recovery replays our journal instead, avoiding duplicate text.
	if d.turn == "" {
		return false
	}
	if id != "" && id != d.nativeSession {
		return false
	}
	if d.profile.Provider == "codex" {
		t := text(params, "turnId")
		if t != "" && d.nativeTurn != "" && t != d.nativeTurn {
			return false
		}
	}
	return true
}
func (d *rpcDriver) onNotification(method string, p map[string]any) {
	if !d.matches(p) {
		return
	}
	if d.profile.Provider == "codex" {
		d.codexEvent(method, p)
		return
	}
	if method == "session/update" {
		d.acpEvent(obj(p["update"]))
		return
	}
	switch method {
	case "cursor/update_todos":
		d.emit(Event{Kind: "item.snapshot", Role: "plan", ItemID: "cursor-plan", Text: planText(arr(p["todos"])), SourceMethod: method})
	case "error":
		d.emit(Event{Kind: "notice", Status: "error", Text: "The native agent reported an error.", SourceMethod: method})
	default:
		// Passive queue/catalog/usage/hook notifications are transport facts,
		// not operator messages. Unknown requests remain explicitly rejected.
		return
	}
}
func (d *rpcDriver) codexEvent(method string, p map[string]any) {
	e := Event{SourceMethod: method, ItemID: text(p, "itemId")}
	switch method {
	case "turn/started":
		d.mu.Lock()
		d.nativeTurn = text(obj(p["turn"]), "id")
		d.mu.Unlock()
		return
	case "turn/completed":
		t := obj(p["turn"])
		d.mu.Lock()
		id := text(t, "id")
		ok := id != "" && (d.nativeTurn == "" || id == d.nativeTurn)
		done := d.turnDone
		d.mu.Unlock()
		if !ok {
			return
		}
		status := "failed"
		switch text(t, "status") {
		case "completed":
			status = "completed"
		case "interrupted":
			status = "cancelled"
		}
		select {
		case done <- nativeTerminal{id: id, status: status, message: nativeErrorMessage(obj(t["error"]))}:
		default:
		}
		return
	case "item/agentMessage/delta":
		e.Kind = "item.delta"
		e.Role = "assistant"
		e.Text = text(p, "delta")
	case "item/plan/delta":
		e.Kind = "item.delta"
		e.Role = "plan"
		e.Text = text(p, "delta")
	case "item/reasoning/summaryTextDelta":
		e.Kind = "item.delta"
		e.Role = "activity"
		e.Text = text(p, "delta")
		e.Details = map[string]any{"type": "reasoningSummary"}
		copyNumber(e.Details, p, "summaryIndex", true)
	case "item/commandExecution/outputDelta":
		e.Kind = "item.delta"
		e.Role = "tool"
		e.Text = text(p, "delta")
	case "item/started", "item/completed":
		item := obj(p["item"])
		e.ItemID = text(item, "id")
		e.Kind = "item.snapshot"
		e.Status = text(item, "status")
		e.Details = codexItemDetails(item)
		switch text(item, "type") {
		case "userMessage":
			return // The exact user input was persisted before sending.
		case "agentMessage":
			e.Role = "assistant"
			e.Text = text(item, "text")
		case "plan":
			e.Role = "plan"
			e.Text = text(item, "text")
		case "reasoning":
			return // Never expose raw reasoning blocks.
		case "commandExecution":
			e.Role = "tool"
			e.Title = text(item, "command")
			e.Text = text(item, "aggregatedOutput")
		case "fileChange":
			e.Role = "tool"
			e.Title = "File changes"
			for _, v := range arr(item["changes"]) {
				m := obj(v)
				e.Text += text(m, "path") + "\n" + text(m, "diff") + "\n"
			}
		case "mcpToolCall":
			e.Role = "tool"
			e.Title = text(item, "tool")
			e.Text = contentText(arr(obj(item["result"])["content"]))
		case "webSearch":
			e.Role = "tool"
			e.Title = "Web search"
			e.Text = text(item, "query")
		case "contextCompaction":
			e.Role = "activity"
			e.Title = "Native context compaction"
		default:
			return
		}
		if method == "item/completed" && e.Status == "" {
			e.Status = "completed"
		}
	case "turn/diff/updated":
		e.Kind = "item.snapshot"
		e.ItemID = "turn-diff"
		e.Role = "tool"
		e.Title = "Proposed diff"
		e.Text = text(p, "diff")
	case "error":
		e.Kind = "notice"
		e.Status = "error"
		e.Text = nativeErrorMessage(obj(p["error"]))
		if e.Text == "" {
			e.Text = "The native agent reported an error."
		}
	case "serverRequest/resolved":
		return // rpcPeer has already cancelled the matching native request context.
	case "thread/tokenUsage/updated", "thread/status/changed":
		return
	default:
		if method == "item/reasoning/textDelta" || method == "item/reasoning/summaryPartAdded" {
			return
		}
		return
	}
	if (e.Kind == "item.delta" || e.Kind == "item.snapshot") && e.ItemID == "" {
		e.Kind = "notice"
		e.Text = "Native item omitted its identity; content was not merged."
	}
	d.emit(e)
}

// Retain only the provider's user-facing message, not transport diagnostics.
func nativeErrorMessage(value map[string]any) string {
	message := []rune(strings.TrimSpace(text(value, "message")))
	if len(message) > 4096 {
		message = append(message[:4095], '…')
	}
	return string(message)
}
func (d *rpcDriver) acpEvent(u map[string]any) {
	kind := text(u, "sessionUpdate")
	e := Event{SourceMethod: "session/update:" + kind}
	switch kind {
	case "agent_message_chunk", "agent_thought_chunk", "user_message_chunk":
		if kind == "user_message_chunk" {
			return
		}
		content := obj(u["content"])
		if text(content, "type") != "text" {
			d.emit(Event{Kind: "notice", Text: "Native non-text content is not supported by this renderer.", SourceMethod: e.SourceMethod})
			return
		}
		e.Kind = "item.delta"
		e.Role = "assistant"
		if kind == "agent_thought_chunk" {
			e.Role = "activity"
		}
		d.mu.Lock()
		// ACP messageId is optional. Without it, allocate a new segment at role/tool
		// boundaries instead of merging every reply in a turn into one message.
		id := text(u, "messageId")
		if id != "" {
			d.messageID = id
			d.messageRole = e.Role
		} else if d.messageID == "" || d.messageRole != e.Role {
			d.messageSerial++
			d.messageID = fmt.Sprintf("segment-%d", d.messageSerial)
			d.messageRole = e.Role
		}
		e.ItemID = d.messageID
		d.mu.Unlock()
		e.Text = text(content, "text")
	case "tool_call", "tool_call_update":
		d.mu.Lock()
		d.messageID = ""
		d.messageRole = ""
		d.mu.Unlock()
		e.Kind = "item.patch"
		e.Role = "tool"
		e.ItemID = text(u, "toolCallId")
		e.Title = text(u, "title")
		e.Status = text(u, "status")
		if c, ok := u["content"]; ok {
			e.Kind = "item.snapshot"
			e.Text = toolContent(arr(c))
		}
		if e.ItemID == "" {
			d.emit(Event{Kind: "notice", Text: "Native tool omitted its identity."})
			return
		}
	case "plan":
		e.Kind = "item.snapshot"
		e.Role = "plan"
		e.ItemID = "plan"
		e.Text = planText(arr(u["entries"]))
	case "available_commands_update", "current_mode_update", "config_option_update", "session_info_update", "usage_update":
		return
	case "error":
		e.Kind = "notice"
		e.Status = "error"
		e.Text = "The native agent reported an error."
	default:
		return
	}
	d.emit(e)
}
func contentText(content []any) string {
	result := ""
	for _, v := range content {
		m := obj(v)
		if text(m, "type") == "text" {
			result += text(m, "text") + "\n"
		}
	}
	return result
}
func toolContent(content []any) string {
	result := ""
	for _, v := range content {
		m := obj(v)
		switch text(m, "type") {
		case "content":
			result += contentText([]any{m["content"]})
		case "diff":
			result += text(m, "path") + "\n--- before\n" + text(m, "oldText") + "\n+++ after\n" + text(m, "newText") + "\n"
		default:
			result += "[native non-text tool content]\n"
		}
	}
	return result
}
func planText(entries []any) string {
	result := ""
	for _, v := range entries {
		m := obj(v)
		result += "[" + text(m, "status") + "] " + text(m, "content") + "\n"
	}
	return result
}
