package nativeagent

import (
	"context"
	"errors"
	"testing"
	"time"
)

type queuedCall struct {
	text    string
	options NativeOptions
}
type queueDriver struct {
	conversationDriver
	started chan queuedCall
	finish  chan string
}

func (d *queueDriver) PromptWithOptions(ctx context.Context, turn, prompt string, options NativeOptions) error {
	d.started <- queuedCall{prompt, options}
	select {
	case status := <-d.finish:
		return d.sink(Event{Kind: "turn.end", TurnID: turn, Status: status})
	case <-ctx.Done():
		return ctx.Err()
	}
}
func queueBroker(t *testing.T) (*Broker, *queueDriver, Profile) {
	b, p, _ := conversationBroker(t, false)
	d := &queueDriver{started: make(chan queuedCall, 20), finish: make(chan string, 20)}
	b.factory = func(_ Profile, sink Sink, ask Ask) (Driver, error) { d.sink = sink; d.ask = ask; return d, nil }
	return b, d, p
}
func nextQueued(t *testing.T, d *queueDriver, want string) {
	t.Helper()
	select {
	case got := <-d.started:
		if got.text != want {
			t.Fatalf("sent %q want %q", got.text, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("queue did not advance")
	}
}
func TestPromptQueueOrdersEditsRemovesAndPausesAfterFailure(t *testing.T) {
	b, d, _ := queueBroker(t)
	s := createConversation(t, b, "queue")
	put := func(key, text string) PromptQueue {
		q, e := b.Queue(s.ID, key, QueueCommand{Operation: "enqueue", Text: text})
		if e != nil {
			t.Fatal(e)
		}
		return q
	}
	put("one", "first")
	nextQueued(t, d, "first")
	put("two", "second")
	put("three", "third")
	q := put("four", "fourth")
	if len(q.Items) != 3 {
		t.Fatalf("queue=%+v", q)
	}
	if _, e := b.Queue(s.ID, "bad", QueueCommand{Operation: "remove", ID: q.Items[0].ID, ExpectedRevision: q.Revision - 1}); !errors.Is(e, ErrConflict) {
		t.Fatalf("stale edit=%v", e)
	}
	q, e := b.Queue(s.ID, "edit", QueueCommand{Operation: "edit", ID: q.Items[1].ID, Text: "third edited", ExpectedRevision: q.Revision})
	if e != nil {
		t.Fatal(e)
	}
	q, e = b.Queue(s.ID, "remove", QueueCommand{Operation: "remove", ID: q.Items[2].ID, ExpectedRevision: q.Revision})
	if e != nil {
		t.Fatal(e)
	}
	q, e = b.Queue(s.ID, "sort", QueueCommand{Operation: "reorder", Order: []string{q.Items[1].ID, q.Items[0].ID}, ExpectedRevision: q.Revision})
	if e != nil {
		t.Fatal(e)
	}
	d.finish <- "completed"
	nextQueued(t, d, "third edited")
	d.finish <- "failed"
	waitState(t, b, s.ID, "ready")
	live := b.sessions[s.ID]
	live.mu.Lock()
	q = cloneQueue(live.queue)
	live.mu.Unlock()
	if !q.Paused || len(q.Items) != 1 {
		t.Fatalf("not paused=%+v", q)
	}
	select {
	case <-d.started:
		t.Fatal("failure drained queue")
	default:
	}
	_, e = b.Queue(s.ID, "resume", QueueCommand{Operation: "resume", ExpectedRevision: q.Revision})
	if e != nil {
		t.Fatal(e)
	}
	nextQueued(t, d, "second")
	d.finish <- "completed"
	waitState(t, b, s.ID, "ready")
	put("two", "second")
	select {
	case <-d.started:
		t.Fatal("idempotent submission replayed")
	default:
	}
}
func TestQueuedMessagesSurviveRestartWithoutAutomaticReplay(t *testing.T) {
	b, _, p := queueBroker(t)
	s := createConversation(t, b, "persist")
	_, e := b.Queue(s.ID, "pause", QueueCommand{Operation: "pause"})
	if e != nil {
		t.Fatal(e)
	}
	q, e := b.Queue(s.ID, "saved", QueueCommand{Operation: "enqueue", Text: "saved message"})
	if e != nil {
		t.Fatal(e)
	}
	b.Close()
	restored, e := NewBroker(b.root, []Profile{p}, b.prepare, b.factory)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	got := restored.sessions[s.ID].queue
	if !got.Paused || len(got.Items) != 1 || got.Items[0].ID != q.Items[0].ID {
		t.Fatalf("lost queue=%+v", got)
	}
	live := restored.sessions[s.ID]
	live.mu.Lock()
	e = live.appendLocked(Event{Kind: "item.snapshot", TurnID: got.Items[0].ID, ItemID: "user", Role: "user", Text: got.Items[0].Text, Status: "submitted", Details: promptDetails(NativeOptions{})})
	live.mu.Unlock()
	if e != nil {
		t.Fatal(e)
	}
	restored.Close()
	again, e := NewBroker(b.root, []Profile{p}, b.prepare, b.factory)
	if e != nil {
		t.Fatal(e)
	}
	defer again.Close()
	if len(again.sessions[s.ID].queue.Items) != 0 {
		t.Fatal("submitted message returned to queue")
	}
}
