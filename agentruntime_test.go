package agentruntime

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// bg is the context of calls whose cancellation a test does not exercise.
var bg = context.Background()

// TestMain doubles as a deterministic, real subprocess protocol fixture. It
// never contacts a vendor, reads credentials, or claims a live subscription run.
func TestMain(m *testing.M) {
	// Metadata probes must never recursively run the test suite.
	if len(os.Args) > 1 && os.Args[1] == "about" {
		fmt.Println(`{"version":"fixture"}`)
		os.Exit(0)
	}
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("native-fixture 1.0.0")
		os.Exit(0)
	}
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("--mcp-config --strict-mcp-config --input-format --append-system-prompt")
		os.Exit(0)
	}
	if len(os.Args) > 1 && (os.Args[1] == "acp" || os.Args[1] == "-c" || os.Args[1] == "--no-auto-update" || os.Args[1] == "--print") {
		fixtureProcess()
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func fixtureProcess() {
	if os.Args[1] == "--print" {
		for _, arg := range os.Args {
			if arg == "--mcp-config" {
				fixtureClaudeMCP()
				return
			}
		}
		for _, arg := range os.Args {
			if arg == "--input-format" {
				fixtureClaudeControl()
				return
			}
		}
		_, _ = io.ReadAll(os.Stdin)
		fmt.Println(`{"type":"system","subtype":"init","session_id":"claude-fixture"}`)
		fmt.Println(`{"type":"stream_event","event":{"type":"message_start","message":{"id":"m1"}}}`)
		fmt.Println(`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}}`)
		fmt.Println(`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}}`)
		fmt.Println(`{"type":"assistant","message":{"id":"m1","content":[{"type":"text","text":"hello"}]}}`)
		fmt.Println(`{"type":"result","subtype":"success","is_error":false,"session_id":"claude-fixture","result":"hello"}`)
		return
	}
	acp := os.Args[1] != "-c"
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), MaxFrame)
	send := func(id any, method string, params any, result any) {
		m := map[string]any{}
		if acp {
			m["jsonrpc"] = "2.0"
		}
		if id != nil {
			m["id"] = id
		}
		if method != "" {
			m["method"] = method
			m["params"] = params
		} else {
			m["result"] = result
		}
		b, _ := json.Marshal(m)
		fmt.Println(string(b))
	}
	var promptID any
	for scanner.Scan() {
		var m map[string]any
		if json.Unmarshal(scanner.Bytes(), &m) != nil {
			return
		}
		id, method := m["id"], text(m, "method")
		switch method {
		case "initialize":
			send(id, "", nil, map[string]any{"protocolVersion": 1, "agentCapabilities": map[string]any{"loadSession": true}, "authMethods": []any{map[string]any{"id": "cached_token"}}})
		case "initialized":
		case "cursor/list_available_models":
			send(id, "", nil, map[string]any{"models": []any{}})
		case "session/set_model", "session/set_config_option", "session/set_mode":
			send(id, "", nil, map[string]any{})
		case "model/list":
			send(id, "", nil, map[string]any{"data": []any{map[string]any{"model": "fixture-native", "displayName": "Native configured model", "defaultReasoningEffort": "medium", "supportedReasoningEfforts": []any{map[string]any{"reasoningEffort": "medium"}}}, map[string]any{"model": "fixture-luna", "displayName": "Fixture Luna", "defaultReasoningEffort": "low", "supportedReasoningEfforts": []any{map[string]any{"reasoningEffort": "low"}}}}})
		case "config/read":
			send(id, "", nil, map[string]any{"config": map[string]any{"model": "fixture-native", "model_reasoning_effort": "medium", "secret": "never-surface-this"}})
		case "account/read":
			send(id, "", nil, map[string]any{"account": map[string]any{"type": "chatgpt"}})
		case "authenticate":
			send(id, "", nil, map[string]any{})
		case "session/new", "session/load":
			send(id, "", nil, map[string]any{"sessionId": "native-fixture"})
		case "thread/start", "thread/resume":
			send(id, "", nil, map[string]any{"thread": map[string]any{"id": "native-fixture"}})
		case "session/prompt":
			promptID = id
			send(nil, "session/update", map[string]any{"sessionId": "native-fixture", "update": map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]any{"type": "text", "text": "hello"}}}, nil)
			send("approve-1", "session/request_permission", map[string]any{"sessionId": "native-fixture", "toolCall": map[string]any{"toolCallId": "tool-1", "title": "Read fixture"}, "options": []any{map[string]any{"optionId": "allow", "kind": "allow_once", "name": "Allow"}, map[string]any{"optionId": "deny", "kind": "reject_once", "name": "Deny"}}}, nil)
		case "turn/start":
			params := obj(m["params"])
			inputs := arr(params["input"])
			if len(inputs) > 0 && text(obj(inputs[0]), "text") == "require-options" {
				if text(params, "model") != "fixture-luna" || text(params, "effort") != "low" || text(params, "approvalPolicy") != "untrusted" || text(obj(params["sandboxPolicy"]), "type") != "readOnly" {
					return
				}
			}
			send(nil, "turn/started", map[string]any{"threadId": "native-fixture", "turn": map[string]any{"id": "native-turn"}}, nil)
			send(id, "", nil, map[string]any{"turn": map[string]any{"id": "native-turn"}})
			send(nil, "item/agentMessage/delta", map[string]any{"threadId": "native-fixture", "turnId": "native-turn", "itemId": "message", "delta": "hello"}, nil)
			send("approve-1", "item/commandExecution/requestApproval", map[string]any{"threadId": "native-fixture", "turnId": "native-turn", "itemId": "tool-1", "command": "read fixture"}, nil)
		case "session/cancel":
			if promptID != nil {
				send(promptID, "", nil, map[string]any{"stopReason": "cancelled"})
				promptID = nil
			}
		case "turn/interrupt":
			send(id, "", nil, map[string]any{})
			send(nil, "turn/completed", map[string]any{"threadId": "native-fixture", "turn": map[string]any{"id": "native-turn", "status": "interrupted"}}, nil)
		case "":
			if id == "approve-1" {
				if acp {
					send(nil, "session/update", map[string]any{"sessionId": "native-fixture", "update": map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": "tool-1", "status": "completed"}}, nil)
					if promptID != nil {
						send(promptID, "", nil, map[string]any{"stopReason": "end_turn"})
						promptID = nil
					}
				} else {
					send(nil, "item/completed", map[string]any{"threadId": "native-fixture", "turnId": "native-turn", "item": map[string]any{"id": "message", "type": "agentMessage", "text": "hello"}}, nil)
					send(nil, "turn/completed", map[string]any{"threadId": "native-fixture", "turn": map[string]any{"id": "native-turn", "status": "completed"}}, nil)
				}
			}
		}
	}
}
func testProfile(t *testing.T, provider string) Profile {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return Profile{ID: provider, Provider: provider, Executable: path, SHA256: hex.EncodeToString(sum[:]), ReviewedVersion: "protocol-fixture-only", BillingReviewed: true}
}
func scope(profile string) Create {
	return Create{ProfileID: profile, Context: ContextRef{Kind: "fixture", ID: "project"}, Workspace: WorkspaceRef{ID: "workspace", Revision: strings.Repeat("a", 40)}, AgentAccessConfirmed: true}
}
func testBroker(t *testing.T, provider string) (*Broker, Session) {
	t.Helper()
	root := t.TempDir()
	b, err := NewBroker(testStorage(t), []Profile{testProfile(t, provider)}, func(ctx context.Context, c Create, id string, fresh bool) (string, error) { return root, ctx.Err() }, nil, testNativeFiles(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Close() })
	s, err := b.Create(context.Background(), "create-1", scope(provider))
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
	return b, s
}
func waitState(t *testing.T, b *Broker, id, state string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s, e := b.Get(bg, id)
		if e != nil {
			t.Fatal(e)
		}
		if s.State == state {
			return
		}
		time.Sleep(time.Millisecond * 5)
	}
	s, _ := b.Get(bg, id)
	events, _, _ := b.replay(id, 0)
	t.Fatalf("state=%s expected=%s events=%+v", s.State, state, events)
}
func waitRequest(t *testing.T, b *Broker, id string) Request {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		events, _, err := b.replay(id, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range events {
			if e.Kind == "input.request" {
				return *e.Request
			}
		}
		time.Sleep(time.Millisecond * 5)
	}
	t.Fatal("no input request")
	return Request{}
}

func TestFiveNativeProviderSubprocessContracts(t *testing.T) {
	for _, provider := range []string{"codex", "cursor", "opencode", "grok", "claude"} {
		t.Run(provider, func(t *testing.T) {
			b, s := testBroker(t, provider)
			turn, err := b.Prompt(bg, s.ID, "prompt-1", PromptRequest{Text: "hello"})
			if err != nil {
				t.Fatal(err)
			}
			if provider != "claude" {
				r := waitRequest(t, b, s.ID)
				if err = b.Answer(bg, s.ID, r.ID, Answer{OptionID: r.Options[0].ID}); err != nil {
					t.Fatal(err)
				}
			}
			waitState(t, b, s.ID, "ready")
			events, _, err := b.replay(s.ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			textSeen, endSeen := false, false
			for _, e := range events {
				if e.TurnID == turn && e.Kind == "item.delta" && e.Text == "hello" {
					textSeen = true
				}
				if e.Kind == "turn.end" && e.Status == "completed" {
					endSeen = true
				}
			}
			if !textSeen || !endSeen {
				t.Fatalf("missing stream or terminal: %+v", events)
			}
		})
	}
}
func TestCommandReplayConflictAndBusySession(t *testing.T) {
	b, s := testBroker(t, "opencode")
	a, err := b.Prompt(bg, s.ID, "p1", PromptRequest{Text: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	waitRequest(t, b, s.ID)
	same, err := b.Prompt(bg, s.ID, "p1", PromptRequest{Text: "hello"})
	if err != nil || same != a {
		t.Fatal("exact prompt replay failed")
	}
	if _, err = b.Prompt(bg, s.ID, "p1", PromptRequest{Text: "changed"}); err != ErrConflict {
		t.Fatalf("conflict=%v", err)
	}
	if _, err = b.Prompt(bg, s.ID, "p2", PromptRequest{Text: "new"}); err != ErrUnavailable {
		t.Fatalf("busy=%v", err)
	}
	again, err := b.Create(context.Background(), "create-1", scope("opencode"))
	if err != nil || again.ID != s.ID {
		t.Fatal("create replay failed")
	}
	changed := scope("opencode")
	changed.Workspace.ID = "other"
	if _, err = b.Create(context.Background(), "create-1", changed); err != ErrConflict {
		t.Fatal("create drift accepted")
	}
}
func TestConcurrentPromptReplayCreatesOneTurn(t *testing.T) {
	b, s := testBroker(t, "opencode")
	var wg sync.WaitGroup
	errors := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := b.Prompt(bg, s.ID, "same", PromptRequest{Text: "one prompt"})
			errors <- err
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	events, _, _ := b.replay(s.ID, 0)
	users := 0
	for _, e := range events {
		if e.Role == "user" {
			users++
		}
	}
	if users != 1 {
		t.Fatalf("user count=%d", users)
	}
}
func TestPermissionValidationIsolationReplayAndCancel(t *testing.T) {
	b, s := testBroker(t, "cursor")
	_, _ = b.Prompt(bg, s.ID, "p1", PromptRequest{Text: "hello"})
	r := waitRequest(t, b, s.ID)
	if b.Answer(bg, s.ID, r.ID, Answer{OptionID: "allow_always"}) == nil {
		t.Fatal("unoffered decision accepted")
	}
	if b.Answer(bg, "other-session", r.ID, Answer{Cancel: true}) != ErrNotFound {
		t.Fatal("cross session decision accepted")
	}
	answer := Answer{OptionID: r.Options[1].ID}
	if err := b.Answer(bg, s.ID, r.ID, answer); err != nil {
		t.Fatal(err)
	}
	if err := b.Answer(bg, s.ID, r.ID, answer); err != nil {
		t.Fatal(err)
	}
	if err := b.Answer(bg, s.ID, r.ID, Answer{Cancel: true}); err != ErrConflict {
		t.Fatalf("decision drift=%v", err)
	}
	waitState(t, b, s.ID, "ready")
}
func TestNativeCancellationAndNoSilentCompletion(t *testing.T) {
	b, s := testBroker(t, "grok")
	_, _ = b.Prompt(bg, s.ID, "p1", PromptRequest{Text: "hello"})
	r := waitRequest(t, b, s.ID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.Cancel(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
	if err := b.Answer(bg, s.ID, r.ID, Answer{OptionID: "allow"}); err != ErrStale {
		t.Fatalf("stale answer=%v", err)
	}
	events, _, _ := b.replay(s.ID, 0)
	found := false
	for _, e := range events {
		if e.Kind == "turn.end" && e.Status == "cancelled" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing confirmed cancellation")
	}
}
func TestJournalRestartReplaysWithoutPromptResend(t *testing.T) {
	b, s := testBroker(t, "claude")
	_, _ = b.Prompt(bg, s.ID, "p1", PromptRequest{Text: "hello"})
	waitState(t, b, s.ID, "ready")
	events, _, _ := b.replay(s.ID, 0)
	cursor := events[len(events)-1].Seq
	root, profiles, prepare := b.store, []Profile{testProfile(t, "claude")}, b.prepare
	b.Close()
	recovered, err := NewBroker(root, profiles, prepare, nil, b.ctx.Value(nativeFilesKey{}).(*NativeFiles))
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Close()
	snapshot, _ := recovered.Get(bg, s.ID)
	if snapshot.State != "disconnected" {
		t.Fatalf("restart state=%s", snapshot.State)
	}
	turn, err := recovered.Prompt(bg, s.ID, "p1", PromptRequest{Text: "hello"})
	if err != nil || turn == "" {
		t.Fatalf("durable replay=%v", err)
	}
	remaining, _, err := recovered.replay(s.ID, cursor)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range remaining {
		if e.Role == "user" {
			t.Fatal("restart resent user prompt")
		}
	}
	if _, _, err = recovered.replay(s.ID, 99999); err != ErrCursor {
		t.Fatal("ahead cursor accepted")
	}
	if err = recovered.Reconnect(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	waitState(t, recovered, s.ID, "ready")
}
func TestConfigAndEnvironmentNeverAcceptCredentials(t *testing.T) {
	got := AgentEnvironment([]string{"HOME=/private/home", "PATH=/bin", "OPENAI_API_KEY=secret", "ANTHROPIC_AUTH_TOKEN=secret", "XAI_API_KEY=secret", "CURSOR_API_KEY=secret", "CODEX_HOME=/other", "RESEARCH_OS_SECRET=secret"})
	if len(got) != 1 {
		t.Fatalf("environment=%v", got)
	}
	for _, provider := range []string{"codex", "cursor", "grok", "opencode", "claude"} {
		args, _ := commandArgs(provider)
		all := strings.Join(args, " ")
		for _, bad := range []string{"bypass", "skip-permissions", "always-approve", "--api-key"} {
			if strings.Contains(all, bad) {
				t.Fatal("unsafe argv")
			}
		}
	}
	p := testProfile(t, "codex")
	p.SHA256 = strings.Repeat("0", 64)
	if checkExecutable(p) == nil {
		t.Fatal("digest drift accepted")
	}
	if _, err := decodeConfigBytes([]byte(`{"schemaVersion":"` + Schema + `","profiles":[],"apiKey":"secret"}`)); err == nil {
		t.Fatal("unknown credential configuration accepted")
	}
}
func TestRequestSchemasCursorQuestionsAndPlans(t *testing.T) {
	var request Request
	driver := &rpcDriver{profile: Profile{Provider: "cursor"}, nativeSession: "n", turn: "t", turnContext: context.Background(), sink: func(Event) error { return nil }, ask: func(ctx context.Context, r Request) (Answer, error) {
		request = r
		if r.Kind == "question" {
			return Answer{Answers: map[string][]string{"q": {"a", "b"}}}, nil
		}
		return Answer{OptionID: "rejected"}, nil
	}}
	result, err := driver.onRequest(context.Background(), "cursor/ask_question", map[string]any{"questions": []any{map[string]any{"id": "q", "prompt": "choose", "allowMultiple": true, "options": []any{map[string]any{"id": "a", "label": "A"}, map[string]any{"id": "b", "label": "B"}}}}})
	if err != nil || text(obj(obj(result)["outcome"]), "outcome") != "answered" || !request.Questions[0].Multiple {
		t.Fatalf("question result=%v err=%v", result, err)
	}
	result, err = driver.onRequest(context.Background(), "cursor/create_plan", map[string]any{"plan": "inspect first"})
	if err != nil || text(obj(obj(result)["outcome"]), "outcome") != "rejected" {
		t.Fatalf("plan=%v %v", result, err)
	}
	if _, err = driver.onRequest(context.Background(), "terminal/create", map[string]any{}); err == nil {
		t.Fatal("unadvertised terminal execution accepted")
	}
}
func TestACPOptionalMessageIDsAndToolStatusPatches(t *testing.T) {
	events := []Event{}
	d := &rpcDriver{turn: "t", sink: func(e Event) error { events = append(events, e); return nil }}
	chunk := func(textValue string) {
		d.acpEvent(map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]any{"type": "text", "text": textValue}})
	}
	chunk("a")
	chunk("b")
	d.acpEvent(map[string]any{"sessionUpdate": "tool_call", "toolCallId": "tool", "title": "Read", "content": []any{map[string]any{"type": "content", "content": map[string]any{"type": "text", "text": "body"}}}})
	d.acpEvent(map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": "tool", "status": "completed"})
	chunk("c")
	if events[0].ItemID != events[1].ItemID || events[0].ItemID == events[4].ItemID || events[3].Kind != "item.patch" {
		t.Fatalf("events=%+v", events)
	}
}
func TestCodexRejectsCrossThreadAndUsesAuthoritativeSnapshot(t *testing.T) {
	events := []Event{}
	d := &rpcDriver{profile: Profile{Provider: "codex"}, turn: "t", nativeSession: "n", nativeTurn: "nt", sink: func(e Event) error { events = append(events, e); return nil }}
	d.onNotification("item/agentMessage/delta", map[string]any{"threadId": "other", "turnId": "nt", "itemId": "i", "delta": "bad"})
	d.onNotification("item/agentMessage/delta", map[string]any{"threadId": "n", "turnId": "nt", "itemId": "i", "delta": "draft"})
	d.onNotification("item/completed", map[string]any{"threadId": "n", "turnId": "nt", "item": map[string]any{"id": "i", "type": "agentMessage", "text": "final"}})
	if len(events) != 2 || events[0].Kind != "item.delta" || events[1].Kind != "item.snapshot" || events[1].Text != "final" {
		t.Fatalf("events=%+v", events)
	}
}
func TestClaudeFinalSnapshotAndPermissionDenial(t *testing.T) {
	events := []Event{}
	d := claudeDecoder{turn: "t", blocks: map[int]claudeBlock{}, sink: func(e Event) error { events = append(events, e); return nil }}
	lines := []string{`{"type":"assistant","session_id":"c","message":{"id":"m","content":[{"type":"text","text":"final"}]}}`, `{"type":"result","session_id":"c","subtype":"success","is_error":false,"result":"final","permission_denials":[{"tool_name":"Bash"}]}`}
	for _, line := range lines {
		var m map[string]any
		_ = json.Unmarshal([]byte(line), &m)
		if err := d.consume(m); err != nil {
			t.Fatal(err)
		}
	}
	if !d.terminal || d.status != "blocked" {
		t.Fatal("denial reported as successful research")
	}
	messages := 0
	for _, e := range events {
		if e.Role == "assistant" {
			messages++
		}
	}
	if messages != 1 {
		t.Fatal("final message duplicated")
	}
}
func TestBodyAndEventCursorGuards(t *testing.T) {
	b, s := testBroker(t, "claude")
	handler := Handler(b, HandlerOptions{})
	req := func(method, path, contentType, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if contentType != "" {
			r.Header.Set("Content-Type", contentType)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	for name, tc := range map[string]struct {
		method, path, contentType, body string
		status                          int
		code                            string
	}{
		"arbitrary command input":  {"POST", "/sessions", "application/json", `{"executable":"/bin/sh"}`, 400, "invalid_argument"},
		"form encoded body":        {"POST", "/sessions", "application/x-www-form-urlencoded", `profileId=claude`, 400, "invalid_argument"},
		"trailing JSON":            {"POST", "/sessions/" + s.ID + "/cancel", "application/json", `{} {}`, 202, ""},
		"unknown session":          {"GET", "/sessions/s_missing", "", "", 404, "not_found"},
		"unknown route":            {"GET", "/sessions/" + s.ID + "/missing", "", "", 404, "not_found"},
		"wrong method":             {"GET", "/sessions/" + s.ID + "/cancel", "", "", 405, "invalid_argument"},
		"oversized page limit":     {"GET", "/sessions?limit=101", "", "", 400, "invalid_argument"},
		"non numeric page limit":   {"GET", "/sessions?limit=many", "", "", 400, "invalid_argument"},
		"events of a missing item": {"GET", "/sessions/s_missing/events", "", "", 404, "not_found"},
	} {
		t.Run(name, func(t *testing.T) {
			w := req(tc.method, tc.path, tc.contentType, tc.body)
			if tc.code == "" {
				return // a body-less route does not read its body
			}
			var envelope struct {
				Error struct{ Code, Message string }
			}
			if w.Code != tc.status || json.Unmarshal(w.Body.Bytes(), &envelope) != nil || envelope.Error.Code != tc.code || envelope.Error.Message == "" {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
func TestMissingStorageBindingFailsClosed(t *testing.T) {
	if b, err := NewBroker(nil, nil, func(context.Context, Create, string, bool) (string, error) { return t.TempDir(), nil }, nil); err == nil {
		b.Close()
		t.Fatal("missing storage accepted")
	}
	if _, err := decodeConfigBytes([]byte(`{"schemaVersion":"xgc.native-agent/v1","profiles":[]}`)); err == nil {
		t.Fatal("retired schema accepted")
	}
}
func TestEventStreamDisconnectDoesNotCancelWorker(t *testing.T) {
	b, s := testBroker(t, "cursor")
	_, _ = b.Prompt(bg, s.ID, "p1", PromptRequest{Text: "hello"})
	waitRequest(t, b, s.ID)
	ctx, cancel := context.WithCancel(context.Background())
	r := httptest.NewRequest("GET", "/sessions/"+s.ID+"/events?after=0", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { Handler(b, HandlerOptions{}).ServeHTTP(w, r); close(done) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done
	if !strings.Contains(w.Body.String(), "event: event\n") || !strings.Contains(w.Body.String(), "id: 1\n") {
		t.Fatalf("no real event frames: %q", w.Body.String())
	}
	snapshot, _ := b.Get(bg, s.ID)
	if snapshot.State != "awaiting-input" {
		t.Fatalf("disconnect changed worker: %s", snapshot.State)
	}
}

func TestUpstreamResolvedRequestExpiresWithoutLateReply(t *testing.T) {
	inputReader, inputWriter := io.Pipe()
	outputReader, outputWriter := io.Pipe()
	defer inputReader.Close()
	defer outputWriter.Close()
	started, expired := make(chan struct{}), make(chan struct{})
	peer := newPeer(inputWriter, outputReader, false, func(string, map[string]any) {}, func(ctx context.Context, method string, p map[string]any) (any, error) {
		close(started)
		<-ctx.Done()
		close(expired)
		return map[string]any{"decision": "accept"}, nil
	})
	defer peer.Close()
	fmt.Fprintln(outputWriter, `{"id":"request-1","method":"item/commandExecution/requestApproval","params":{"threadId":"thread"}}`)
	<-started
	fmt.Fprintln(outputWriter, `{"method":"serverRequest/resolved","params":{"threadId":"wrong","requestId":"request-1"}}`)
	select {
	case <-expired:
		t.Fatal("cross-thread resolution cancelled request")
	case <-time.After(10 * time.Millisecond):
	}
	fmt.Fprintln(outputWriter, `{"method":"serverRequest/resolved","params":{"threadId":"thread","requestId":"request-1"}}`)
	select {
	case <-expired:
	case <-time.After(time.Second):
		t.Fatal("request did not expire")
	}
}

func TestShutdownCancelsPendingWorkspacePreparation(t *testing.T) {
	preparing := make(chan struct{})
	b, err := NewBroker(testStorage(t), []Profile{testProfile(t, "codex")}, func(ctx context.Context, c Create, id string, fresh bool) (string, error) {
		close(preparing)
		<-ctx.Done()
		return "", ctx.Err()
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	s, err := b.Create(context.Background(), "shutdown-create", scope("codex"))
	if err != nil {
		t.Fatal(err)
	}
	<-preparing
	done := make(chan struct{})
	go func() { b.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown waited for handshake timeout")
	}
	final, _ := b.Get(bg, s.ID)
	if final.State != "disconnected" {
		t.Fatalf("state after shutdown: %s", final.State)
	}
	if _, err = b.Prompt(bg, s.ID, "later", PromptRequest{Text: "must not start"}); err == nil {
		t.Fatal("accepted prompt after shutdown")
	}
	if err = b.Reconnect(bg, s.ID); err == nil {
		t.Fatal("accepted reconnect after shutdown")
	}
}

func fixtureClaudeControl() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var m map[string]any
		if json.Unmarshal(scanner.Bytes(), &m) != nil {
			return
		}
		switch text(m, "type") {
		case "control_request":
			if text(obj(m["request"]), "subtype") == "initialize" {
				fmt.Println(`{"type":"control_response","response":{"subtype":"success","request_id":"xgc-initialize","response":{}}}`)
			}
		case "user":
			fmt.Println(`{"type":"system","subtype":"init","session_id":"claude-control-fixture"}`)
			fmt.Println(`{"type":"control_request","request_id":"permission-1","request":{"subtype":"can_use_tool","tool_name":"Bash","tool_use_id":"tool-1","input":{"command":"fixture-only"},"description":"Run the fixture operation"}}`)
		case "control_response":
			response := obj(obj(m["response"])["response"])
			value := "denied"
			if text(response, "behavior") == "allow" {
				if text(obj(response["updatedInput"]), "command") != "fixture-only" {
					return
				}
				value = "allowed"
			}
			fmt.Println(`{"type":"stream_event","event":{"type":"message_start","message":{"id":"message-1"}}}`)
			fmt.Println(`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}}`)
			data, _ := json.Marshal(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": value}}})
			fmt.Println(string(data))
			data, _ = json.Marshal(map[string]any{"type": "result", "subtype": "success", "is_error": false, "session_id": "claude-control-fixture", "result": value})
			fmt.Println(string(data))
			return
		}
	}
}
