package agentruntime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
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
	// stop is the turn the operator asked to stop. It is remembered until the
	// native client can take it (the native turn is acknowledged, the ACP prompt
	// is on the wire) and delivered once: stopSent is the turn it went out for.
	stop       string
	stopSent   string
	promptSent bool
	// tools is what subscribers have been sent of each streamed tool output of
	// the turn (see toolUpdate).
	tools map[string]*streamedTool
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
	c, err := startChild(ctx, d.profile, args, cwd, environment...)
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
		_, err = d.peer.Call(handshake, "initialize", map[string]any{"clientInfo": map[string]any{"name": "xgc-agent-runtime", "title": "XGC", "version": "0.1.0"}, "capabilities": map[string]any{"experimentalApi": false}})
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
			return errors.New("ChatGPT login required; API billing is not used")
		}
		params := map[string]any{"cwd": cwd}
		localMCPThreadContext(ctx, params)
		if err = codexThreadOptions(params, d.profile.Defaults); err != nil {
			return err
		}
		if nativeID != "" {
			params["threadId"] = nativeID
			// Only the session metadata is consumed; excluding turns keeps resume cheap
			// and tolerant of historical item schema drift (t3code CodexSessionRuntime).
			params["excludeTurns"] = true
			result, err = d.resumeThread(handshake, nativeID, params)
		} else {
			result, err = d.peer.Call(handshake, "thread/start", params)
		}
		if err == nil {
			nativeID = text(obj(result["thread"]), "id")
		}
	} else {
		capabilities := map[string]any{"fs": map[string]any{"readTextFile": false, "writeTextFile": false}, "terminal": false}
		if d.profile.Provider == "cursor" {
			capabilities["_meta"] = map[string]any{"parameterizedModelPicker": true}
		}
		init, e := d.peer.Call(handshake, "initialize", acpInitializeParams(d.profile.Provider, capabilities))
		if e != nil {
			return e
		}
		if version, ok := init["protocolVersion"].(float64); !ok || version != 1 {
			return errors.New("unsupported ACP protocol version")
		}
		if d.profile.Provider == "cursor" {
			if _, err = d.peer.Call(handshake, "authenticate", map[string]any{"methodId": "cursor_login"}); err != nil {
				return errors.New("the existing Cursor login is unavailable")
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
				return errors.New("Grok cached login is unavailable; log in with the Grok client first")
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
				return errors.New("this ACP version cannot resume a session")
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
		return errors.New("session identity missing")
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
	d.promptSent = false
	d.tools = map[string]*streamedTool{}
	native := d.nativeSession
	done := d.turnDone
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		d.turnContext = nil
		d.turn = ""
		d.nativeTurn = ""
		d.promptSent = false
		d.mu.Unlock()
	}()
	if d.stopRequested(turn) {
		return d.stoppedBeforeStart()
	}
	if d.profile.Provider != "codex" {
		if err := d.applyACPOptions(ctx, options); err != nil {
			return err
		}
		if d.stopRequested(turn) {
			return d.stoppedBeforeStart()
		}
		wait, err := d.peer.start(ctx, "session/prompt", map[string]any{"sessionId": native, "prompt": localMCPACPContent(ctx, prompt)})
		if err != nil {
			return err
		}
		// A stop that came while the prompt was being written would have reached
		// the client before it: send it now that the prompt is on the wire.
		if err = d.deliverStop(ctx, turn, true); err != nil {
			d.emit(Event{Kind: "notice", Status: "error", Text: stopNotDelivered})
		}
		result, err := wait()
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
		d.flushTools()
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
		return errors.New("turn identity missing")
	}
	d.mu.Lock()
	if d.nativeTurn == "" {
		d.nativeTurn = nativeTurn
	}
	same := d.nativeTurn == nativeTurn
	d.mu.Unlock()
	if !same {
		return errors.New("turn identity mismatch")
	}
	// A stop that came before the native turn was acknowledged is applied now.
	if err = d.deliverStop(ctx, turn, false); err != nil {
		d.emit(Event{Kind: "notice", Status: "error", Text: stopNotDelivered})
	}
	for {
		select {
		case terminal := <-done:
			if terminal.id != nativeTurn {
				continue
			}
			d.flushTools()
			d.emit(Event{Kind: "turn.end", Status: terminal.status, Text: terminal.message, SourceMethod: "turn/completed"})
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-d.peer.done:
			return io.EOF
		}
	}
}

const stopNotDelivered = "The stop request could not be delivered to the client."

// Cancel records the stop and delivers it when the native client can take it:
// at once for a running turn, otherwise when the turn starts.
func (d *rpcDriver) Cancel(ctx context.Context, turn string) error {
	d.mu.Lock()
	d.stop = turn
	peer := d.peer
	d.mu.Unlock()
	if peer == nil {
		return ErrUnavailable
	}
	return d.deliverStop(ctx, turn, false)
}

func (d *rpcDriver) stopRequested(turn string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stop == turn
}

// stoppedBeforeStart ends a turn the operator stopped before it reached the
// native client; nothing was sent, so nothing needs to be interrupted.
func (d *rpcDriver) stoppedBeforeStart() error {
	d.emit(Event{Kind: "turn.end", Status: "cancelled", SourceMethod: "stop"})
	return nil
}

// deliverStop sends the remembered stop of turn to the native client, once.
// Codex interrupts the native turn, which exists only after turn/started or the
// turn/start result; ACP cancels the session, which only means something while
// its prompt is on the wire. ACP's prompt marks that moment with sent.
func (d *rpcDriver) deliverStop(ctx context.Context, turn string, sent bool) error {
	d.mu.Lock()
	if sent {
		d.promptSent = true
	}
	due := d.stop == turn && d.stopSent != turn && d.turn == turn
	if d.profile.Provider == "codex" {
		due = due && d.nativeTurn != ""
	} else {
		due = due && d.promptSent
	}
	if due {
		d.stopSent = turn
	}
	native, nativeTurn := d.nativeSession, d.nativeTurn
	d.mu.Unlock()
	if !due {
		return nil
	}
	if d.profile.Provider == "codex" {
		_, err := d.peer.Call(ctx, "turn/interrupt", map[string]any{"threadId": native, "turnId": nativeTurn})
		return err
	}
	cancel := map[string]any{"sessionId": native}
	if d.profile.Provider == "grok" {
		cancel["_meta"] = grokCancelMeta
	}
	return d.peer.Notify("session/cancel", cancel)
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
		<-process.done
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
		d.emit(Event{Kind: "notice", Status: "error", Text: "The client reported an error.", SourceMethod: method})
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
		turn, turnContext := d.turn, d.turnContext
		d.mu.Unlock()
		if turnContext != nil && d.stopRequested(turn) {
			// The read loop must not wait for the response of its own request.
			go func() {
				if d.deliverStop(turnContext, turn, false) != nil {
					d.emit(Event{Kind: "notice", Status: "error", Text: stopNotDelivered})
				}
			}()
		}
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
			e.Title = "Context compaction"
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
		var emit bool
		if e, emit = d.toolUpdate(e, text(p, "diff")); !emit {
			return
		}
	case "error":
		e.Kind = "notice"
		e.Status = "error"
		e.Text = nativeErrorMessage(obj(p["error"]))
		if e.Text == "" {
			e.Text = "The client reported an error."
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
		e.Text = "This item omitted its identity; content was not merged."
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
			d.emit(Event{Kind: "notice", Text: "Non-text content is not supported by this renderer.", SourceMethod: e.SourceMethod})
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
		// A tool call that appears ends the message in progress. Progress on a call
		// already shown, such as a background command finishing, does not: the
		// answer around it stays one message.
		d.mu.Lock()
		if _, shown := d.tools[text(u, "toolCallId")]; !shown {
			d.messageID = ""
			d.messageRole = ""
		}
		d.mu.Unlock()
		e.Kind = "item.patch"
		e.Role = "tool"
		e.ItemID = text(u, "toolCallId")
		e.Title = text(u, "title")
		e.Status = text(u, "status")
		if e.ItemID == "" {
			d.emit(Event{Kind: "notice", Text: "This tool omitted its identity."})
			return
		}
		if c, ok := u["content"]; ok {
			e.Kind = "item.snapshot"
			var emit bool
			if e, emit = d.toolUpdate(e, toolContent(arr(c))); !emit {
				return
			}
		} else {
			d.toolPatched(e)
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
		e.Text = "The client reported an error."
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

// toolContent renders the content of a tool call. Entries are separated, not
// terminated, by a line break, so output that grows only appends to the text.
func toolContent(content []any) string {
	parts := []string{}
	for _, v := range content {
		m := obj(v)
		switch text(m, "type") {
		case "content":
			parts = append(parts, strings.TrimSuffix(contentText([]any{m["content"]}), "\n"))
		case "diff":
			parts = append(parts, text(m, "path")+"\n--- before\n"+text(m, "oldText")+"\n+++ after\n"+text(m, "newText"))
		default:
			parts = append(parts, "[non-text tool content]")
		}
	}
	return strings.Join(parts, "\n")
}
func planText(entries []any) string {
	result := ""
	for _, v := range entries {
		m := obj(v)
		result += "[" + text(m, "status") + "] " + text(m, "content") + "\n"
	}
	return result
}

// streamedToolPersistEvery is how many rewrites of a tool's output are folded
// into one event when the output did not merely grow.
const streamedToolPersistEvery = 10

// streamedTool is what subscribers have been sent of one tool's output.
type streamedTool struct {
	text, title, status string
	// big output is no longer compared with what was sent, to bound memory.
	big bool
	// latest is the newest output a rewrite held back, sent at the end of the turn at the latest.
	latest  string
	skipped int
}

// maxStreamedTool bounds the output kept per tool to find what was appended.
const maxStreamedTool = 1 << 20

// toolUpdate decides what a repeated full snapshot of a tool's output costs.
// Agents resend the whole output with every update, so journaling each one
// grows the log with the square of the output. Nothing is sent when the output
// did not change; output that only grew is sent as the appended suffix; other
// rewrites are sent every streamedToolPersistEvery-th time. The first sight of
// a tool, a new title or status, and a terminal status are always sent in full.
func (d *rpcDriver) toolUpdate(e Event, output string) (Event, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.tools == nil {
		d.tools = map[string]*streamedTool{}
	}
	tool := d.tools[e.ItemID]
	first := tool == nil
	if first {
		tool = &streamedTool{}
		d.tools[e.ItemID] = tool
	}
	moved := first || (e.Title != "" && e.Title != tool.title) || (e.Status != "" && e.Status != tool.status)
	terminal := e.Status == "completed" || e.Status == "failed"
	if e.Title != "" {
		tool.title = e.Title
	}
	if e.Status != "" {
		tool.status = e.Status
	}
	if !moved && !terminal {
		switch {
		case tool.skipped == 0 && !tool.big && output == tool.text:
			return e, false
		case tool.skipped == 0 && !tool.big && strings.HasPrefix(output, tool.text):
			e.Kind, e.Title, e.Status, e.Text = "item.delta", "", "", output[len(tool.text):]
			tool.text = output
			tool.big = len(output) > maxStreamedTool
			return e, true
		case tool.skipped+1 < streamedToolPersistEvery:
			tool.skipped++
			tool.latest = output
			return e, false
		}
	}
	tool.text, tool.big = output, len(output) > maxStreamedTool
	if tool.big {
		tool.text = ""
	}
	tool.latest, tool.skipped = "", 0
	e.Text = output
	return e, true
}

// toolPatched records the title and status a patch (an update without content) sent.
func (d *rpcDriver) toolPatched(e Event) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.tools == nil {
		d.tools = map[string]*streamedTool{}
	}
	tool := d.tools[e.ItemID]
	if tool == nil {
		tool = &streamedTool{}
		d.tools[e.ItemID] = tool
	}
	if e.Title != "" {
		tool.title = e.Title
	}
	if e.Status != "" {
		tool.status = e.Status
	}
}

// flushTools sends the output that rewrites held back, so a turn never ends
// with a tool showing less than the agent last said.
func (d *rpcDriver) flushTools() {
	d.mu.Lock()
	held := []Event{}
	for id, tool := range d.tools {
		if tool.skipped > 0 {
			held = append(held, Event{Kind: "item.snapshot", Role: "tool", ItemID: id, Text: tool.latest})
			tool.text, tool.big = tool.latest, len(tool.latest) > maxStreamedTool
			if tool.big {
				tool.text = ""
			}
			tool.latest, tool.skipped = "", 0
		}
	}
	d.mu.Unlock()
	sort.Slice(held, func(i, j int) bool { return held[i].ItemID < held[j].ItemID })
	for _, e := range held {
		d.emit(e)
	}
}

var archivedThread = regexp.MustCompile(`(?i)\bsession \S+ is archived\b|\bcodex unarchive\b`)

// resumeThread resumes a Codex thread. A thread the client archived is
// unarchived and resumed again, which keeps its history (t3code #15389); a
// client without excludeTurns is asked again for the full history.
func (d *rpcDriver) resumeThread(ctx context.Context, threadID string, params map[string]any) (map[string]any, error) {
	result, err := d.peer.Call(ctx, "thread/resume", params)
	if err == nil {
		return result, nil
	}
	message := nativeMessage(err)
	switch {
	case archivedThread.MatchString(message):
		if _, unarchived := d.peer.Call(ctx, "thread/unarchive", map[string]any{"threadId": threadID}); unarchived != nil {
			return nil, err
		}
		return d.peer.Call(ctx, "thread/resume", params)
	case strings.Contains(message, "excludeTurns"):
		delete(params, "excludeTurns")
		return d.peer.Call(ctx, "thread/resume", params)
	}
	return nil, err
}
