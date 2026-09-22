package agentruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// Legacy callers retain the narrow Read/Glob/Grep stream. Configured providers
// use the native stream-json control channel in claude_options.go; operator
// choices are forwarded to the native CLI without an additional agent loop.
type claudeDriver struct {
	profile       Profile
	sink          Sink
	ask           Ask
	mu            sync.Mutex
	process       *child
	cwd           string
	nativeSession string
	cancelled     bool
}

func (d *claudeDriver) Open(ctx context.Context, cwd, nativeID string) error {
	if err := checkExecutable(d.profile); err != nil {
		return err
	}
	if err := checkClaudeLocalMCP(ctx, d.profile); err != nil {
		return err
	}
	d.cwd = cwd
	d.nativeSession = nativeID
	return nil
}
func (d *claudeDriver) Prompt(ctx context.Context, turn, prompt string) error {
	if _, bound := localMCP(ctx); bound {
		// Retain the legacy built-in tool restriction while allowing the native
		// CLI to ask permission for the separately reviewed MCP tools.
		return d.promptWithControl(ctx, turn, prompt, AgentOptions{}, true)
	}
	args, _ := commandArgs("claude")
	d.mu.Lock()
	native, cwd := d.nativeSession, d.cwd
	d.mu.Unlock()
	if native != "" {
		args = append(args, "--resume="+native)
	}
	c, err := startChild(d.profile, args, cwd)
	if err != nil {
		return err
	}
	d.mu.Lock()
	d.process = c
	d.cancelled = false
	d.mu.Unlock()
	stop := context.AfterFunc(ctx, func() { c.Stop() })
	defer stop()
	// Prompt bytes go through stdin, never the process list.
	go func() { _, _ = c.stdin.Write([]byte(prompt)); _ = c.stdin.Close() }()
	decoder := claudeDecoder{turn: turn, native: native, sink: d.sink, blocks: map[int]claudeBlock{}}
	scanner := bufio.NewScanner(c.stdout)
	scanner.Buffer(make([]byte, 4096), MaxFrame)
	for scanner.Scan() {
		var m map[string]any
		if json.Unmarshal(scanner.Bytes(), &m) != nil {
			err = errors.New("invalid Claude CLI event")
			break
		}
		if e := decoder.consume(m); e != nil {
			err = e
			break
		}
	}
	if scanner.Err() != nil {
		err = errors.New("Claude CLI event exceeds transport limits")
	}
	if err != nil {
		go c.Stop()
	}
	waitErr := c.Wait()
	d.mu.Lock()
	d.process = nil
	cancelled := d.cancelled
	if decoder.native != "" {
		d.nativeSession = decoder.native
	}
	d.mu.Unlock()
	if cancelled {
		return d.sink(Event{Kind: "turn.end", TurnID: turn, Status: "cancelled", SourceMethod: "claude:process-exit"})
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return err
	}
	if !decoder.terminal {
		return errors.New("Claude CLI exited without a terminal result; outcome is unknown")
	}
	if waitErr != nil && decoder.status == "completed" {
		return errors.New("Claude CLI exited unsuccessfully after result")
	}
	return d.sink(Event{Kind: "turn.end", TurnID: turn, Status: decoder.status, SourceMethod: "claude:result"})
}
func (d *claudeDriver) Cancel(ctx context.Context) error {
	d.mu.Lock()
	d.cancelled = true
	p := d.process
	d.mu.Unlock()
	if p == nil {
		return ErrStale
	}
	go p.Stop()
	return nil
}
func (d *claudeDriver) Close() error {
	d.mu.Lock()
	p := d.process
	d.mu.Unlock()
	if p != nil {
		p.Stop()
	}
	return nil
}

type claudeBlock struct {
	ID    string
	Role  string
	Title string
}
type claudeDecoder struct {
	turn          string
	native        string
	message       string
	blocks        map[int]claudeBlock
	sink          Sink
	terminal      bool
	status        string
	assistantSeen bool
	lastTextID    string
	authFailed    bool
	rateLimited   bool
}

func (d *claudeDecoder) emit(e Event) error {
	e.TurnID = d.turn
	e.SourceMethod = "claude:stream-json"
	return d.sink(e)
}
func (d *claudeDecoder) consume(m map[string]any) error {
	if d.terminal {
		return nil
	} // No post-result frames can change a finalized turn.
	if id := text(m, "session_id"); id != "" {
		if len(id) > 512 {
			return errors.New("invalid native session ID")
		}
		if d.native != "" && d.native != id {
			return errors.New("Claude session identity mismatch")
		}
		if d.native == "" {
			d.native = id
			if err := d.emit(Event{Kind: "session.identity", AgentSessionID: id}); err != nil {
				return err
			}
		}
	}
	switch text(m, "type") {
	case "system":
		return nil
	case "control_request":
		return errors.New("this Claude CLI requires an unsupported interactive control channel; no approval was sent")
	case "stream_event":
		e := obj(m["event"])
		index := intNumber(e["index"])
		switch text(e, "type") {
		case "message_start":
			d.message = text(obj(e["message"]), "id")
			d.blocks = map[int]claudeBlock{}
		case "content_block_start":
			if d.message == "" {
				return errors.New("Claude block precedes message identity")
			}
			b := obj(e["content_block"])
			block := claudeBlock{ID: fmt.Sprintf("%s:%d", d.message, index), Role: "assistant"}
			switch text(b, "type") {
			case "text":
				d.lastTextID = block.ID
			case "tool_use":
				block.ID = text(b, "id")
				block.Role = "tool"
				block.Title = text(b, "name")
			case "thinking", "redacted_thinking":
				block.Role = "activity"
			default:
				return nil
			}
			d.blocks[index] = block
			if block.Role == "activity" {
				return nil
			}
			return d.emit(Event{Kind: "item.snapshot", ItemID: block.ID, Role: block.Role, Title: block.Title, Text: text(b, "text"), Status: "running"})
		case "content_block_delta":
			block, ok := d.blocks[index]
			if !ok {
				return nil
			}
			delta := obj(e["delta"])
			if text(delta, "type") == "text_delta" {
				return d.emit(Event{Kind: "item.delta", ItemID: block.ID, Role: block.Role, Text: text(delta, "text")})
			}
			if text(delta, "type") == "input_json_delta" {
				return d.emit(Event{Kind: "item.delta", ItemID: block.ID, Role: "tool", Text: text(delta, "partial_json")})
			}
		}
	case "assistant":
		// The CLI can report authentication failure or a rejected usage window on
		// the assistant message before ending the turn as a generic API error;
		// retain that evidence for the result fallback (t3code resultOutcome).
		switch text(m, "error") {
		case "authentication_failed":
			d.authFailed = true
		case "rate_limit":
			d.rateLimited = true
		}
		message := obj(m["message"])
		id := text(message, "id")
		if id == "" {
			return errors.New("Claude assistant identity missing")
		}
		for index, v := range arr(message["content"]) {
			b := obj(v)
			kind := text(b, "type")
			itemID := fmt.Sprintf("%s:%d", id, index)
			if kind == "text" {
				d.assistantSeen = true
				if err := d.emit(Event{Kind: "item.snapshot", ItemID: itemID, Role: "assistant", Text: text(b, "text"), Status: "completed"}); err != nil {
					return err
				}
			}
			if kind == "tool_use" {
				input, _ := json.Marshal(b["input"])
				if err := d.emit(Event{Kind: "item.snapshot", ItemID: text(b, "id"), Role: "tool", Title: text(b, "name"), Text: string(input), Status: "running"}); err != nil {
					return err
				}
			}
		}
	case "user":
		for _, v := range arr(obj(m["message"])["content"]) {
			b := obj(v)
			if text(b, "type") != "tool_result" {
				continue
			}
			body := str(b["content"])
			if body == "" {
				body = contentText(arr(b["content"]))
			}
			status := "completed"
			if b["is_error"] == true {
				status = "failed"
			}
			if err := d.emit(Event{Kind: "item.snapshot", ItemID: text(b, "tool_use_id"), Role: "tool", Text: body, Status: status}); err != nil {
				return err
			}
		}
	case "result":
		d.terminal = true
		d.status = "failed"
		if m["is_error"] != true && text(m, "subtype") == "success" {
			d.status = "completed"
		}
		// Structured terminal evidence outranks the success subtype: the CLI can
		// tag a dead turn success with an empty error list (t3code resultOutcome).
		failure := ""
		if text(m, "subtype") == "success" && intNumber(m["api_error_status"]) == 529 {
			failure = "Claude API is overloaded (529). Try again shortly."
		} else if msg := claudeTerminalReasonError(text(m, "terminal_reason")); msg != "" {
			failure = msg
		} else if text(m, "subtype") == "success" && m["is_error"] == true {
			if d.authFailed {
				failure = "Native Claude login has expired; sign in again with the Claude CLI."
			} else if d.rateLimited {
				failure = "Claude usage limit reached. Send the message again once the limit resets."
			}
		}
		if failure != "" {
			d.status = "failed"
			if err := d.emit(Event{Kind: "notice", Status: "error", Text: failure}); err != nil {
				return err
			}
		}
		if len(arr(m["permission_denials"])) > 0 {
			d.status = "blocked"
			if err := d.emit(Event{Kind: "notice", Status: "blocked", Text: "The native permission policy prevented one or more requested tool operations."}); err != nil {
				return err
			}
		}
		if !d.assistantSeen && text(m, "result") != "" {
			itemID := d.lastTextID
			if itemID == "" {
				itemID = "result"
			}
			return d.emit(Event{Kind: "item.snapshot", ItemID: itemID, Role: "assistant", Text: text(m, "result"), Status: d.status})
		}
	case "error":
		return d.emit(Event{Kind: "notice", Status: "error", Text: "The native Claude client reported an error."})
	default:
		return nil // Passive native telemetry is not an operator message.
	}
	return nil
}
func intNumber(v any) int { n, _ := v.(float64); return int(n) }

// claudeTerminalReasonError names the dead-turn terminal reasons the CLI stamps
// when it gives up, even on success-tagged results (t3code terminalResultError).
func claudeTerminalReasonError(reason string) string {
	switch reason {
	case "api_error":
		return "Claude gave up after repeated API errors."
	case "malformed_tool_use_exhausted":
		return "Claude gave up after repeated malformed tool calls."
	case "budget_exhausted":
		return "Claude stopped after exhausting its budget."
	case "structured_output_retry_exhausted", "tool_deferred_unavailable", "turn_setup_failed",
		"blocking_limit", "rapid_refill_breaker", "prompt_too_long", "image_error", "model_error":
		return "Claude turn failed (" + reason + ")."
	}
	return ""
}
