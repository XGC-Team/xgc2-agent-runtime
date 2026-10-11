package agentruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// stopDriver is a queueDriver whose Stop is observable and can be refused or
// can end the turn the way a native client does.
type stopDriver struct {
	*queueDriver
	stops   chan string
	refuse  error
	endWith string
}

func (d *stopDriver) Cancel(_ context.Context, turn string) error {
	d.stops <- turn
	if d.refuse != nil {
		return d.refuse
	}
	if d.endWith != "" {
		d.finish <- d.endWith
	}
	return nil
}

func stopBroker(t *testing.T) (*Broker, *stopDriver) {
	t.Helper()
	b, q, _ := queueBroker(t)
	d := &stopDriver{queueDriver: q, stops: make(chan string, 4)}
	b.factory = func(_ Profile, sink Sink, ask Ask) (Driver, error) { q.sink, q.ask = sink, ask; return d, nil }
	return b, d
}

func enqueue(t *testing.T, b *Broker, id, key, text string) PromptQueue {
	t.Helper()
	q, err := b.Queue(bg, id, key, QueueCommand{Operation: "enqueue", Text: text})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func queueState(b *Broker, id string) PromptQueue {
	s, _ := b.get(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneQueue(s.queue)
}

func neverStarts(t *testing.T, d *queueDriver) {
	t.Helper()
	select {
	case call := <-d.started:
		t.Fatalf("held queue started %q", call.text)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestStopHoldsTheQueueEvenWhenTheTurnCompletesNormally(t *testing.T) {
	b, d := stopBroker(t)
	s := createConversation(t, b, "stop-hold")
	enqueue(t, b, s.ID, "one", "first")
	nextQueued(t, d.queueDriver, "first")
	enqueue(t, b, s.ID, "two", "second")
	enqueue(t, b, s.ID, "three", "third")

	d.endWith = "completed" // the native turn finishes before it notices the stop
	if err := b.Cancel(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
	if q := queueState(b, s.ID); !q.Paused || len(q.Items) != 2 {
		t.Fatalf("stop did not hold the queue: %+v", q)
	}
	neverStarts(t, d.queueDriver)
	// Reconnecting does not release the hold either.
	q := queueState(b, s.ID)
	if _, err := b.Queue(bg, s.ID, "resume", QueueCommand{Operation: "resume", ExpectedRevision: q.Revision}); err != nil {
		t.Fatal(err)
	}
	nextQueued(t, d.queueDriver, "second")
}

func TestStopHoldsMessagesQueuedWhileTheStopIsInFlight(t *testing.T) {
	b, d := stopBroker(t)
	s := createConversation(t, b, "stop-inflight")
	enqueue(t, b, s.ID, "one", "first")
	nextQueued(t, d.queueDriver, "first")
	if err := b.Cancel(bg, s.ID); err != nil { // the client has accepted the stop and winds down
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "cancelling")
	enqueue(t, b, s.ID, "late", "sent while stopping")
	d.finish <- "cancelled"
	waitState(t, b, s.ID, "ready")
	if q := queueState(b, s.ID); !q.Paused || len(q.Items) != 1 {
		t.Fatalf("message queued during the stop ran: %+v", q)
	}
	neverStarts(t, d.queueDriver)
}

func TestStopOfARunWithAnEmptyQueueDoesNotHoldLaterMessages(t *testing.T) {
	b, d := stopBroker(t)
	s := createConversation(t, b, "stop-empty")
	enqueue(t, b, s.ID, "one", "first")
	nextQueued(t, d.queueDriver, "first")
	d.endWith = "cancelled"
	if err := b.Cancel(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
	if q := queueState(b, s.ID); q.Paused {
		t.Fatalf("an empty queue is held: %+v", q)
	}
	enqueue(t, b, s.ID, "next", "sent after the stop")
	nextQueued(t, d.queueDriver, "sent after the stop")
}

func TestRefusedStopHoldsTheQueueButKeepsTheTurnAndItsApprovalsAlive(t *testing.T) {
	b, d := stopBroker(t)
	d.refuse = errors.New("native client refused the interrupt")
	s := createConversation(t, b, "stop-refused")
	enqueue(t, b, s.ID, "one", "first")
	nextQueued(t, d.queueDriver, "first")
	enqueue(t, b, s.ID, "two", "second")

	// An approval is pending when the stop fails.
	answers := make(chan Answer, 2)
	ask := func(title string) {
		go func() {
			answer, err := d.ask(context.Background(), Request{Kind: "permission", Title: title, Options: []Option{{ID: "yes", Kind: "allow_once"}, {ID: "no", Kind: "reject_once"}}})
			if err != nil {
				answer = Answer{Cancel: true}
			}
			answers <- answer
		}()
	}
	ask("before the stop")
	waitState(t, b, s.ID, "awaiting-input")
	if err := b.Cancel(bg, s.ID); err == nil {
		t.Fatal("a refused stop was reported as accepted")
	}
	if got, _ := b.Get(bg, s.ID); got.State != "awaiting-input" {
		t.Fatalf("a refused stop changed the worker to %s", got.State)
	}
	pending, err := b.Inputs(bg, s.ID)
	if err != nil || len(pending) != 1 {
		t.Fatalf("approval was cancelled by a failed stop: %+v %v", pending, err)
	}
	// An approval raised after the failed stop is offered as well.
	ask("after the stop")
	deadline := time.Now().Add(2 * time.Second)
	for len(pending) != 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
		pending, _ = b.Inputs(bg, s.ID)
	}
	if len(pending) != 2 {
		t.Fatalf("approval raised after a failed stop was dropped: %+v", pending)
	}
	for _, p := range pending {
		if err := b.Answer(bg, s.ID, p.Request.ID, Answer{OptionID: "yes"}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if answer := <-answers; answer.Cancel || answer.OptionID != "yes" {
			t.Fatalf("approval answer was replaced: %+v", answer)
		}
	}
	// The stop was the operator's decision: the queue is held although the turn still runs.
	if q := queueState(b, s.ID); !q.Paused {
		t.Fatalf("refused stop released the queue: %+v", q)
	}
	d.finish <- "completed"
	waitState(t, b, s.ID, "ready")
	neverStarts(t, d.queueDriver)
}

func TestAStopThatTheClientNeverCompletesIsRecoveredByClosingTheRuntime(t *testing.T) {
	b, d := stopBroker(t)
	s := createConversation(t, b, "stalled-stop")
	turn, err := b.Prompt(bg, s.ID, "stalled", PromptRequest{Text: "never ends"})
	if err != nil {
		t.Fatal(err)
	}
	nextQueued(t, d.queueDriver, "never ends")
	enqueue(t, b, s.ID, "later", "after it")
	if err := b.Cancel(bg, s.ID); err != nil { // accepted, but the native turn never ends
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "cancelling")
	if err := b.CloseSession(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "closed")
	// The run unwinds on its own: its turn ends as unknown, and a new runtime can be started.
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := b.Reconnect(bg, s.ID)
		if err == nil {
			break
		}
		if !errors.Is(err, ErrConflict) || time.Now().After(deadline) {
			t.Fatalf("a closed stalled run was not recoverable: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	waitState(t, b, s.ID, "ready")
	events, _, _ := b.replay(s.ID, 0)
	ended := 0
	for _, e := range events {
		if e.Kind == "turn.end" && e.TurnID == turn {
			ended++
			if e.Status != "unknown" {
				t.Fatalf("the turn of a closed runtime must end as unknown: %+v", e)
			}
		}
	}
	if ended != 1 {
		t.Fatalf("turn ended %d times", ended)
	}
	if q := queueState(b, s.ID); !q.Paused || len(q.Items) != 1 {
		t.Fatalf("the stop's hold was lost: %+v", q)
	}
	neverStarts(t, d.queueDriver)
}

func TestRejectedCommandsDoNotHoldTheQueue(t *testing.T) {
	b, d := stopBroker(t)
	s := createConversation(t, b, "validation")
	enqueue(t, b, s.ID, "one", "first")
	nextQueued(t, d.queueDriver, "first")
	enqueue(t, b, s.ID, "two", "second")
	before := queueState(b, s.ID)
	for name, command := range map[string]QueueCommand{
		"blank":     {Operation: "enqueue", Text: " \n"},
		"unknown":   {Operation: "explode"},
		"stale":     {Operation: "pause", ExpectedRevision: before.Revision - 1},
		"not found": {Operation: "remove", ID: "t_missing", ExpectedRevision: before.Revision},
	} {
		if _, err := b.Queue(bg, s.ID, "bad-"+name[:3], command); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	if _, err := b.Prompt(bg, s.ID, "bad-prompt", PromptRequest{Text: " "}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank prompt: %v", err)
	}
	if _, err := b.Prompt(bg, s.ID, "busy", PromptRequest{Text: "while running"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("prompt while running: %v", err)
	}
	if after := queueState(b, s.ID); after.Paused || after.Revision != before.Revision || len(after.Items) != 1 {
		t.Fatalf("rejected commands changed the queue: %+v -> %+v", before, after)
	}
	d.finish <- "completed"
	nextQueued(t, d.queueDriver, "second")
}

func TestPromptRetryAfterReconnectNeverResends(t *testing.T) {
	b, d := stopBroker(t)
	s := createConversation(t, b, "no-resend")
	turn, err := b.Prompt(bg, s.ID, "send-once", PromptRequest{Text: "do it"})
	if err != nil {
		t.Fatal(err)
	}
	nextQueued(t, d.queueDriver, "do it")
	d.finish <- "failed" // the native client broke without confirming the result
	waitState(t, b, s.ID, "ready")
	// The response to the first call was lost: the client retries with the same key.
	for _, step := range []string{"same worker", "after reconnect"} {
		again, err := b.Prompt(bg, s.ID, "send-once", PromptRequest{Text: "do it"})
		if err != nil || again != turn {
			t.Fatalf("%s: retry returned %q, %v", step, again, err)
		}
		neverStarts(t, d.queueDriver)
		if step == "same worker" {
			if err := b.Reconnect(bg, s.ID); err != nil && !errors.Is(err, ErrConflict) {
				t.Fatal(err)
			}
			waitState(t, b, s.ID, "ready")
		}
	}
	if _, err := b.Prompt(bg, s.ID, "send-once", PromptRequest{Text: "changed"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed retry: %v", err)
	}
}

// lostDriver is a queueDriver whose native client dies in the middle of a turn:
// the prompt call fails and no terminal event was ever sent.
type lostDriver struct{ *queueDriver }

func (d *lostDriver) PromptWithOptions(ctx context.Context, turn, prompt string, options AgentOptions) error {
	d.started <- queuedCall{prompt, options}
	select {
	case <-d.finish:
		return io.ErrUnexpectedEOF
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestALostNativeClientEndsTheTurnAsUnknownAndHoldsTheQueue(t *testing.T) {
	b, q, _ := queueBroker(t)
	d := &lostDriver{q}
	b.factory = func(_ Profile, sink Sink, ask Ask) (Driver, error) { q.sink, q.ask = sink, ask; return d, nil }
	s := createConversation(t, b, "lost")
	turn, err := b.Prompt(bg, s.ID, "send-once", PromptRequest{Text: "do it"})
	if err != nil {
		t.Fatal(err)
	}
	nextQueued(t, q, "do it")
	enqueue(t, b, s.ID, "later", "after it")
	q.finish <- "lost"
	waitState(t, b, s.ID, "disconnected")

	events, _, err := b.replay(s.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	var ends []Event
	for _, e := range events {
		if e.Kind == "turn.end" {
			ends = append(ends, e)
		}
	}
	if len(ends) != 1 || ends[0].TurnID != turn || ends[0].Status != "unknown" {
		t.Fatalf("a lost client must end the turn as unknown, once: %+v", ends)
	}
	if got := queueState(b, s.ID); !got.Paused || len(got.Items) != 1 {
		t.Fatalf("a lost client left the queue running: %+v", got)
	}
	// The prompt may have reached the native client: neither a retry nor a reconnect sends it again.
	if again, err := b.Prompt(bg, s.ID, "send-once", PromptRequest{Text: "do it"}); err != nil || again != turn {
		t.Fatalf("retry returned %q, %v", again, err)
	}
	if err := b.Reconnect(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
	neverStarts(t, q)
	// Only the operator releases the hold.
	held := queueState(b, s.ID)
	if _, err := b.Queue(bg, s.ID, "resume", QueueCommand{Operation: "resume", ExpectedRevision: held.Revision}); err != nil {
		t.Fatal(err)
	}
	nextQueued(t, q, "after it")
}

// codexServer plays the native client's side of a Codex app-server connection.
type codexServer struct {
	t        *testing.T
	requests chan map[string]any
	out      io.Writer
	acp      bool
	mu       sync.Mutex
}

func (s *codexServer) send(v map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.acp {
		v["jsonrpc"] = "2.0"
	}
	data, _ := json.Marshal(v)
	_, _ = s.out.Write(append(data, '\n'))
}
func (s *codexServer) reply(request map[string]any, result any) {
	s.send(map[string]any{"id": request["id"], "result": result})
}
func (s *codexServer) next(method string) map[string]any {
	s.t.Helper()
	select {
	case request := <-s.requests:
		if request["method"] != method {
			s.t.Fatalf("got %v, want %s", request["method"], method)
		}
		return request
	case <-time.After(3 * time.Second):
		s.t.Fatalf("no %s request", method)
		return nil
	}
}
func (s *codexServer) silent() {
	s.t.Helper()
	select {
	case request := <-s.requests:
		s.t.Fatalf("unexpected %v", request["method"])
	case <-time.After(60 * time.Millisecond):
	}
}

func codexFixture(t *testing.T) (*rpcDriver, *codexServer, *[]Event, *sync.Mutex) {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	t.Cleanup(func() { inR.Close(); outW.Close() })
	var mu sync.Mutex
	events := []Event{}
	d := &rpcDriver{profile: Profile{Provider: "codex"}, nativeSession: "thread", sink: func(e Event) error { mu.Lock(); events = append(events, e); mu.Unlock(); return nil }}
	d.peer = newPeer(inW, outR, false, d.onNotification, d.onRequest)
	t.Cleanup(func() { d.peer.Close() })
	server := &codexServer{t: t, requests: make(chan map[string]any, 8), out: outW}
	go func() {
		scanner := bufio.NewScanner(inR)
		for scanner.Scan() {
			var m map[string]any
			if json.Unmarshal(scanner.Bytes(), &m) == nil {
				server.requests <- m
			}
		}
	}()
	return d, server, &events, &mu
}

func turnEnds(events *[]Event, mu *sync.Mutex) []string {
	mu.Lock()
	defer mu.Unlock()
	statuses := []string{}
	for _, e := range *events {
		if e.Kind == "turn.end" {
			statuses = append(statuses, e.Status)
		}
	}
	return statuses
}

func TestCodexStopBeforeTheNativeTurnIsAcknowledgedIsAppliedWhenItStarts(t *testing.T) {
	for _, started := range []string{"response", "notification"} {
		t.Run(started, func(t *testing.T) {
			d, server, events, mu := codexFixture(t)
			done := make(chan error, 1)
			go func() { done <- d.PromptWithOptions(context.Background(), "t1", "hello", AgentOptions{}) }()
			start := server.next("turn/start")
			if err := d.Cancel(context.Background(), "t1"); err != nil {
				t.Fatalf("a stop before the native turn exists was refused: %v", err)
			}
			server.silent() // nothing to interrupt yet
			if started == "notification" {
				server.send(map[string]any{"method": "turn/started", "params": map[string]any{"threadId": "thread", "turn": map[string]any{"id": "native-turn"}}})
			} else {
				server.reply(start, map[string]any{"turn": map[string]any{"id": "native-turn"}})
			}
			interrupt := server.next("turn/interrupt")
			if params := obj(interrupt["params"]); params["threadId"] != "thread" || params["turnId"] != "native-turn" {
				t.Fatalf("interrupt=%v", params)
			}
			server.reply(interrupt, map[string]any{})
			if started == "notification" {
				server.reply(start, map[string]any{"turn": map[string]any{"id": "native-turn"}})
			}
			server.send(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": "thread", "turn": map[string]any{"id": "native-turn", "status": "interrupted"}}})
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if got := turnEnds(events, mu); len(got) != 1 || got[0] != "cancelled" {
				t.Fatalf("turn ends=%v", got)
			}
			server.silent() // delivered exactly once
		})
	}
}

func TestCodexStopBeforeTheTurnIsSentNeverStartsIt(t *testing.T) {
	d, server, events, mu := codexFixture(t)
	if err := d.Cancel(context.Background(), "t1"); err != nil {
		t.Fatal(err)
	}
	if err := d.PromptWithOptions(context.Background(), "t1", "hello", AgentOptions{}); err != nil {
		t.Fatal(err)
	}
	server.silent()
	if got := turnEnds(events, mu); len(got) != 1 || got[0] != "cancelled" {
		t.Fatalf("turn ends=%v", got)
	}
	// A stop that belonged to an earlier turn does not touch the next one.
	done := make(chan error, 1)
	go func() { done <- d.PromptWithOptions(context.Background(), "t2", "again", AgentOptions{}) }()
	start := server.next("turn/start")
	server.reply(start, map[string]any{"turn": map[string]any{"id": "native-2"}})
	server.silent()
	server.send(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": "thread", "turn": map[string]any{"id": "native-2", "status": "completed"}}})
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestCodexStopOfARunningTurnInterruptsImmediatelyAndReportsRefusal(t *testing.T) {
	d, server, _, _ := codexFixture(t)
	done := make(chan error, 1)
	go func() { done <- d.PromptWithOptions(context.Background(), "t1", "hello", AgentOptions{}) }()
	server.reply(server.next("turn/start"), map[string]any{"turn": map[string]any{"id": "native-turn"}})
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		d.mu.Lock()
		known := d.nativeTurn != ""
		d.mu.Unlock()
		if known {
			break
		}
	}
	cancelled := make(chan error, 1)
	go func() { cancelled <- d.Cancel(context.Background(), "t1") }()
	interrupt := server.next("turn/interrupt")
	server.send(map[string]any{"id": interrupt["id"], "error": map[string]any{"code": -32000, "message": "no such turn"}})
	if err := <-cancelled; err == nil {
		t.Fatal("a refused interrupt was reported as accepted")
	}
	server.send(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": "thread", "turn": map[string]any{"id": "native-turn", "status": "completed"}}})
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func acpFixture(t *testing.T) (*rpcDriver, *codexServer, *[]Event, *sync.Mutex) {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	t.Cleanup(func() { inR.Close(); outW.Close() })
	var mu sync.Mutex
	events := []Event{}
	d := &rpcDriver{profile: Profile{Provider: "opencode"}, nativeSession: "session", sink: func(e Event) error { mu.Lock(); events = append(events, e); mu.Unlock(); return nil }}
	d.peer = newPeer(inW, outR, true, d.onNotification, d.onRequest)
	t.Cleanup(func() { d.peer.Close() })
	server := &codexServer{t: t, requests: make(chan map[string]any, 8), out: outW, acp: true}
	go func() {
		scanner := bufio.NewScanner(inR)
		for scanner.Scan() {
			var m map[string]any
			if json.Unmarshal(scanner.Bytes(), &m) == nil {
				server.requests <- m
			}
		}
	}()
	return d, server, &events, &mu
}

func TestACPStopBeforeThePromptIsSentNeverSendsIt(t *testing.T) {
	d, server, events, mu := acpFixture(t)
	if err := d.Cancel(context.Background(), "t1"); err != nil {
		t.Fatal(err)
	}
	if err := d.PromptWithOptions(context.Background(), "t1", "hello", AgentOptions{}); err != nil {
		t.Fatal(err)
	}
	server.silent()
	if got := turnEnds(events, mu); len(got) != 1 || got[0] != "cancelled" {
		t.Fatalf("turn ends=%v", got)
	}
}

func TestACPStopOfARunningPromptCancelsTheSessionOnce(t *testing.T) {
	d, server, events, mu := acpFixture(t)
	done := make(chan error, 1)
	go func() { done <- d.PromptWithOptions(context.Background(), "t1", "hello", AgentOptions{}) }()
	prompt := server.next("session/prompt")
	// The prompt is on the wire, so the stop may go out right away.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		d.mu.Lock()
		sent := d.promptSent
		d.mu.Unlock()
		if sent {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if err := d.Cancel(context.Background(), "t1"); err != nil {
		t.Fatal(err)
	}
	cancel := server.next("session/cancel")
	if cancel["params"].(map[string]any)["sessionId"] != "session" {
		t.Fatalf("cancel=%v", cancel)
	}
	if err := d.Cancel(context.Background(), "t1"); err != nil {
		t.Fatal(err)
	}
	server.silent() // a second Stop click does not send a second cancel
	server.reply(prompt, map[string]any{"stopReason": "cancelled"})
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := turnEnds(events, mu); len(got) != 1 || got[0] != "cancelled" {
		t.Fatalf("turn ends=%v", got)
	}
}

func TestClaudeStopBeforeTheProcessStartsIsNotLost(t *testing.T) {
	for _, mode := range []string{"legacy", "control"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			events := []Event{}
			d := &claudeDriver{profile: testProfile(t, "claude"), sink: func(e Event) error { mu.Lock(); events = append(events, e); mu.Unlock(); return nil }}
			if err := d.Cancel(context.Background(), "t1"); err != nil {
				t.Fatalf("a stop before the process exists was refused: %v", err)
			}
			// No native files are granted in this context: launching would fail,
			// so a clean cancelled end proves nothing was launched.
			var err error
			if mode == "legacy" {
				err = d.Prompt(context.Background(), "t1", "hello")
			} else {
				err = d.PromptWithOptions(context.Background(), "t1", "hello", AgentOptions{Permission: "approval-required"})
			}
			if err != nil {
				t.Fatalf("stopped turn failed: %v", err)
			}
			if got := turnEnds(&events, &mu); len(got) != 1 || got[0] != "cancelled" {
				t.Fatalf("turn ends=%v", got)
			}
		})
	}
}

func TestCodexResumesAnArchivedThreadAndFallsBackWithoutExcludeTurns(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		steps         []string
	}{
		{"archived", "session 019a is archived; run codex unarchive", []string{"thread/resume", "thread/unarchive", "thread/resume"}},
		{"old client", "unknown field `excludeTurns`", []string{"thread/resume", "thread/resume"}},
		{"unrelated failure", "the model is unavailable", []string{"thread/resume"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, server, _, _ := codexFixture(t)
			params := map[string]any{"threadId": "native", "excludeTurns": true}
			done := make(chan error, 1)
			var result map[string]any
			go func() {
				var err error
				result, err = d.resumeThread(context.Background(), "native", params)
				done <- err
			}()
			first := server.next("thread/resume")
			server.send(map[string]any{"id": first["id"], "error": map[string]any{"code": -32600, "message": tc.message}})
			for _, step := range tc.steps[1:] {
				request := server.next(step)
				if step == "thread/unarchive" {
					if obj(request["params"])["threadId"] != "native" {
						t.Fatalf("unarchive=%v", request)
					}
					server.reply(request, map[string]any{})
					continue
				}
				if _, kept := obj(request["params"])["excludeTurns"]; kept == (tc.name == "old client") {
					t.Fatalf("excludeTurns in the retry: %v", request["params"])
				}
				server.reply(request, map[string]any{"thread": map[string]any{"id": "native"}})
			}
			err := <-done
			if tc.name == "unrelated failure" {
				if err == nil || strings.Contains(err.Error(), tc.message) {
					t.Fatalf("err=%v", err)
				}
				server.silent()
				return
			}
			if err != nil || obj(result["thread"])["id"] != "native" {
				t.Fatalf("result=%v err=%v", result, err)
			}
		})
	}
}

func TestClaudeStopAsksTheClientToEndTheTurnItself(t *testing.T) {
	var mu sync.Mutex
	events := []Event{}
	asked := make(chan struct{}, 1)
	driver, err := NewDriver(testProfile(t, "claude"), func(e Event) error { mu.Lock(); events = append(events, e); mu.Unlock(); return nil },
		func(ctx context.Context, r Request) (Answer, error) {
			asked <- struct{}{}
			<-ctx.Done() // the operator never answers: the turn is stopped instead
			return Answer{Cancel: true}, ctx.Err()
		})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	ctx, cancel := context.WithTimeout(testNativeContext(t), 10*time.Second)
	defer cancel()
	if err = driver.Open(ctx, t.TempDir(), ""); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- driver.(optionDriver).PromptWithOptions(ctx, "turn", "fixture", AgentOptions{Permission: "approval-required"})
	}()
	select {
	case <-asked:
	case <-time.After(5 * time.Second):
		t.Fatal("the turn never reached the client")
	}
	started := time.Now()
	if err = driver.Cancel(ctx, "turn"); err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if took := time.Since(started); took > claudeInterruptGrace-500*time.Millisecond {
		t.Fatalf("the turn needed %s: the client was killed instead of interrupted", took)
	}
	if got := turnEnds(&events, &mu); len(got) != 1 || got[0] != "cancelled" {
		t.Fatalf("turn ends=%v", got)
	}
	// Claude ended the turn itself, so the conversation is saved and can be resumed.
	identities := 0
	for _, e := range events {
		if e.Kind == "session.identity" && e.AgentSessionID == "claude-control-fixture" {
			identities++
		}
	}
	if identities != 1 || driver.(*claudeDriver).nativeSession != "claude-control-fixture" {
		t.Fatalf("identity events=%d resume id=%q", identities, driver.(*claudeDriver).nativeSession)
	}
}

func TestClaudeConversationCanBeResumedOnlyOnceATurnEnded(t *testing.T) {
	var identities []string
	decoder := claudeDecoder{turn: "t", blocks: map[int]claudeBlock{}, sink: func(e Event) error {
		if e.Kind == "session.identity" {
			identities = append(identities, e.AgentSessionID)
		}
		return nil
	}}
	feed := func(line string) {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatal(err)
		}
		if err := decoder.consume(m); err != nil {
			t.Fatal(err)
		}
	}
	feed(`{"type":"system","subtype":"init","session_id":"fresh"}`)
	feed(`{"type":"stream_event","session_id":"fresh","event":{"type":"message_start","message":{"id":"m"}}}`)
	if len(identities) != 0 || decoder.native != "fresh" {
		t.Fatalf("a conversation Claude may not have saved was announced: %v", identities)
	}
	feed(`{"type":"result","subtype":"success","is_error":false,"session_id":"fresh","result":"done"}`)
	feed(`{"type":"result","subtype":"success","is_error":false,"session_id":"fresh","result":"again"}`)
	if len(identities) != 1 || identities[0] != "fresh" {
		t.Fatalf("identities=%v", identities)
	}
	// A conversation that was resumed is already known and announces nothing.
	identities = nil
	known := claudeDecoder{turn: "t", native: "earlier", blocks: map[int]claudeBlock{}, sink: decoder.sink}
	line := map[string]any{"type": "result", "subtype": "success", "session_id": "earlier"}
	if err := known.consume(line); err != nil || len(identities) != 0 {
		t.Fatalf("resumed conversation announced %v (%v)", identities, err)
	}
}

func TestGrokModesMetadataAndStop(t *testing.T) {
	for permission, want := range map[string]string{
		"":                  "--no-auto-update agent stdio",
		"approval-required": "--no-auto-update --permission-mode default agent stdio",
		"auto":              "--no-auto-update --permission-mode auto agent stdio",
		"full-access":       "--no-auto-update agent --always-approve stdio",
		// Grok never had an accept-edits launch mode: a stored value launches asking.
		"auto-accept-edits": "--no-auto-update --permission-mode default agent stdio",
	} {
		args, err := grokPermissionArgs(permission)
		if err != nil || strings.Join(args, " ") != want {
			t.Fatalf("%q launches %v (%v), want %s", permission, args, err, want)
		}
	}
	if _, err := grokPermissionArgs("plan"); err == nil {
		t.Fatal("unknown Grok permission accepted")
	}
	advertised := []string{}
	for _, p := range grokPermissions() {
		advertised = append(advertised, p.ID)
	}
	if strings.Join(advertised, ",") != "approval-required,auto,full-access" {
		t.Fatalf("advertised=%v", advertised)
	}
	if params := acpInitializeParams("grok", map[string]any{}); obj(params["_meta"])["clientType"] != "extension" {
		t.Fatalf("grok initialize=%v", params)
	}
	if _, present := acpInitializeParams("opencode", map[string]any{})["_meta"]; present {
		t.Fatal("another provider received Grok's metadata")
	}

	d, server, _, _ := acpFixture(t)
	d.profile = Profile{Provider: "grok"}
	done := make(chan error, 1)
	go func() { done <- d.PromptWithOptions(context.Background(), "t1", "hello", AgentOptions{}) }()
	prompt := server.next("session/prompt")
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		d.mu.Lock()
		sent := d.promptSent
		d.mu.Unlock()
		if sent {
			break
		}
	}
	if err := d.Cancel(context.Background(), "t1"); err != nil {
		t.Fatal(err)
	}
	cancel := server.next("session/cancel")
	if params := obj(cancel["params"]); params["sessionId"] != "session" || obj(params["_meta"])["cancelTrigger"] != "ctrl_c" {
		t.Fatalf("cancel=%v", params)
	}
	server.reply(prompt, map[string]any{"stopReason": "cancelled"})
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
