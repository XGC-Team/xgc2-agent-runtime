package agentruntime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/XGC-Team/xgc2-storage/api"
)

// countedStorage counts the storage calls of a store and can stall or fail its commits.
type countedStorage struct {
	StorageClient
	batches   atomic.Int64
	snapshots atomic.Int64
	gate      chan struct{} // when set, commits wait for it to be closed
	failure   error         // when set, commits fail with it
}

func (c *countedStorage) Snapshot(ctx context.Context, id string, r api.SnapshotRequest) (api.SnapshotResponse, error) {
	c.snapshots.Add(1)
	return c.StorageClient.Snapshot(ctx, id, r)
}
func (c *countedStorage) Batch(ctx context.Context, r api.BatchRequest) (api.Receipt, error) {
	c.batches.Add(1)
	if c.gate != nil {
		select {
		case <-c.gate:
		case <-ctx.Done():
			return api.Receipt{}, ctx.Err()
		}
	}
	if c.failure != nil {
		return api.Receipt{}, c.failure
	}
	return c.StorageClient.Batch(ctx, r)
}

func counted(s *liveSession) *countedStorage {
	c := &countedStorage{StorageClient: s.store.Client}
	s.store.Client = c
	return c
}

func appendDeltas(t testing.TB, s *liveSession, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		s.mu.Lock()
		err := s.appendLocked(Event{Kind: "item.delta", TurnID: "t_stream", ItemID: "message", Role: "assistant", Text: fmt.Sprint(i, ",")})
		s.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestStreamedEventsAreCommittedInFewBatchesWithoutSnapshotReads(t *testing.T) {
	s := testSession(t, "s_batches")
	c := counted(s)
	appendDeltas(t, s, 1000)
	commit(t, s)
	if c.batches.Load() > 8 || c.snapshots.Load() != 0 {
		t.Fatalf("1000 deltas cost %d commits and %d snapshot reads", c.batches.Load(), c.snapshots.Load())
	}
	events := stored(t, s, 0, 1000)
	for i, e := range events {
		if e.Seq != uint64(i+1) || e.Text != fmt.Sprint(i, ",") {
			t.Fatalf("event %d: %+v", i, e)
		}
	}
	rows, err := s.store.records(bg)
	if err != nil || len(rows) != 1 || rows[0].LastSeq != 1000 || rows[0].version != s.version {
		t.Fatalf("record %+v version %q: %v", rows, s.version, err)
	}
}

func TestEventsWaitForCompanyOnlyForTheJournalDelay(t *testing.T) {
	s := testSessionDelay(t, "s_delay", 80*time.Millisecond)
	c := counted(s)
	started := time.Now()
	appendDeltas(t, s, 3)
	time.Sleep(20 * time.Millisecond)
	s.mu.Lock()
	early := s.durable
	s.mu.Unlock()
	if early != 0 || c.batches.Load() != 0 {
		t.Fatalf("committed %d events after %s, before the delay", early, time.Since(started))
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		durable := s.durable
		s.mu.Unlock()
		if durable == 3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if c.batches.Load() != 1 || time.Since(started) > time.Second {
		t.Fatalf("3 deltas took %d commits and %s", c.batches.Load(), time.Since(started))
	}
	// A durable event does not wait for the delay.
	started = time.Now()
	s.mu.Lock()
	_ = s.appendLocked(Event{Kind: "prompt.queue", Queue: &PromptQueue{Revision: 1}})
	seq := s.info.LastSeq
	s.mu.Unlock()
	if err := s.waitDurable(bg, seq); err != nil || time.Since(started) > 60*time.Millisecond {
		t.Fatalf("durable event waited %s: %v", time.Since(started), err)
	}
}

func TestStorageCallsNeverHoldTheBrokerOrSessionLocks(t *testing.T) {
	b, _, _ := conversationBroker(t, false)
	live := createConversation(t, b, "locks")
	session, _ := b.get(live.ID)
	c := counted(session)
	c.gate = make(chan struct{})
	released := false
	release := func() {
		if !released {
			released = true
			close(c.gate)
		}
	}
	defer release()

	// Commits are stalled: streamed output keeps flowing and is readable.
	for i := 0; i < 20; i++ {
		if err := b.receive(session, Event{Kind: "item.delta", TurnID: "", ItemID: "m", Role: "assistant", Text: "x"}); err != nil && !errors.Is(err, ErrStale) {
			t.Fatal(err)
		}
	}
	// A prompt waits for its message to be durable; a create waits for its record.
	blocked := make(chan error, 2)
	go func() { _, err := b.Prompt(bg, live.ID, "waits", PromptRequest{Text: "hello"}); blocked <- err }()
	go func() { _, err := b.Create(bg, "waits-too", scope("fixture")); blocked <- err }()
	deadline := time.Now().Add(2 * time.Second)
	for c.batches.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if c.batches.Load() < 1 {
		t.Fatal("no commit reached storage")
	}

	reads := make(chan string, 8)
	go func() {
		_, err := b.Get(bg, live.ID)
		reads <- fmt.Sprint("get ", err)
		_, err = b.List(bg, SessionListOptions{})
		reads <- fmt.Sprint("list ", err)
		_, err = b.Inputs(bg, live.ID)
		reads <- fmt.Sprint("inputs ", err)
		_, err = b.Attention(bg)
		reads <- fmt.Sprint("attention ", err)
		ctx, cancel := context.WithTimeout(bg, 200*time.Millisecond)
		defer cancel()
		got := 0
		_ = b.Subscribe(ctx, live.ID, 0, func(Event) error { got++; return nil })
		reads <- fmt.Sprint("subscribe ", got > 0)
		reads <- "done"
	}()
	for {
		select {
		case line := <-reads:
			if line == "done" {
				goto freed
			}
			if line == "get <nil>" || line == "list <nil>" || line == "inputs <nil>" || line == "attention <nil>" || line == "subscribe true" {
				continue
			}
			t.Fatalf("read failed while storage was stalled: %s", line)
		case <-time.After(3 * time.Second):
			t.Fatal("reads blocked behind a stalled storage call")
		}
	}
freed:
	release()
	for i := 0; i < 2; i++ {
		select {
		case err := <-blocked:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("stalled commands did not finish after storage came back")
		}
	}
}

// committedEvents reads what storage holds of a conversation, without asking the broker.
func committedEvents(t testing.TB, store *Store, id string) []Event {
	t.Helper()
	rows, err := store.records(bg)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.ID == id {
			events, err := store.readEvents(id, 0, row.LastSeq)
			if err != nil {
				t.Fatal(err)
			}
			return events
		}
	}
	return nil
}

// observingDriver reports what storage holds when the broker acts on a prompt
// and when the native client learns that its turn end was taken.
type observingDriver struct {
	*queueDriver
	t        testing.TB
	b        *Broker
	id       string
	atPrompt chan []Event
	identity chan bool
	atEnd    chan []Event
}

func (d *observingDriver) PromptWithOptions(_ context.Context, turn, _ string, _ AgentOptions) error {
	_, found, err := d.b.store.prompt(d.id, turn)
	if err != nil {
		return err
	}
	d.identity <- found
	d.atPrompt <- committedEvents(d.t, d.b.store, d.id)
	if err = d.sink(Event{Kind: "item.delta", TurnID: turn, ItemID: "m", Role: "assistant", Text: "streamed"}); err != nil {
		return err
	}
	err = d.sink(Event{Kind: "turn.end", TurnID: turn, Status: "completed"})
	d.atEnd <- committedEvents(d.t, d.b.store, d.id)
	return err
}

func TestPromptAndTurnEndAreDurableBeforeTheyAreActedOn(t *testing.T) {
	b, q, _ := queueBroker(t)
	s := createConversation(t, b, "durable")
	d := &observingDriver{queueDriver: q, t: t, b: b, id: s.ID, atPrompt: make(chan []Event, 1), identity: make(chan bool, 1), atEnd: make(chan []Event, 1)}
	b.factory = func(_ Profile, sink Sink, ask Ask) (Driver, error) { q.sink, q.ask = sink, ask; return d, nil }
	// The conversation was created before the factory changed: reconnect to use the observer.
	if err := b.CloseSession(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	if err := b.Reconnect(bg, s.ID); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
	turn, err := b.Prompt(bg, s.ID, "p-durable", PromptRequest{Text: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if found := <-d.identity; !found {
		t.Fatal("the prompt identity was not stored when the native client received the prompt")
	}
	user := false
	for _, e := range <-d.atPrompt {
		user = user || e.TurnID == turn && e.Role == "user" && e.Status == "submitted"
	}
	if !user {
		t.Fatal("the user message was not stored when the native client received the prompt")
	}
	committed := <-d.atEnd
	if last := committed[len(committed)-1]; last.Kind != "turn.end" || last.Status != "completed" {
		t.Fatalf("turn end not stored when the sink returned; last stored event %+v", last)
	}
	waitState(t, b, s.ID, "ready")
}

func TestSubscribersSeeEveryEventExactlyOnceAcrossCommits(t *testing.T) {
	s := testSessionDelay(t, "s_subscribe", 5*time.Millisecond)
	live := &Broker{sessions: map[string]*liveSession{s.id: s}}
	live.ctx, live.cancel = context.WithCancel(bg)
	defer live.cancel()
	const total = 3000
	received := make(chan uint64, total+8)
	ctx, stop := context.WithCancel(bg)
	defer stop()
	done := make(chan error, 1)
	go func() {
		done <- live.Subscribe(ctx, s.id, 0, func(e Event) error { received <- e.Seq; return nil })
	}()
	for i := 0; i < total; i += 100 {
		appendDeltas(t, s, 100)
		time.Sleep(time.Millisecond) // lets commits interleave with the appends
	}
	for want := uint64(1); want <= total; want++ {
		select {
		case got := <-received:
			if got != want {
				t.Fatalf("event %d delivered as %d", want, got)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("subscriber stalled at %d", want)
		}
	}
	stop()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("subscription ended with %v", err)
	}
	// A late subscriber reads the committed part from storage and the rest from the journal.
	commit(t, s)
	appendDeltas(t, s, 5)
	var late []uint64
	ctx, stop = context.WithTimeout(bg, 300*time.Millisecond)
	defer stop()
	_ = live.Subscribe(ctx, s.id, 2990, func(e Event) error { late = append(late, e.Seq); return nil })
	if len(late) != 15 || late[0] != 2991 || late[14] != 3005 {
		t.Fatalf("late subscriber got %v", late)
	}
}

func TestAFailedCommitStopsTheSessionAndReleasesEveryWaiter(t *testing.T) {
	s := testSession(t, "s_failure")
	c := counted(s)
	c.failure = errors.New("disk full")
	appendDeltas(t, s, 3)
	waiters := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { waiters <- s.flushAll(bg) }()
	}
	for i := 0; i < 2; i++ {
		select {
		case err := <-waiters:
			if err == nil || !errors.Is(err, c.failure) {
				t.Fatalf("waiter got %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("a waiter was left behind")
		}
	}
	if err := s.appendLocked(Event{Kind: "notice", Text: "after"}); err == nil {
		t.Fatal("a stopped session took another event")
	}
	if s.info.State != "disconnected" || s.durable != 0 {
		t.Fatalf("state=%s durable=%d", s.info.State, s.durable)
	}
	// Subscribers still read what happened, then see the failure.
	live := &Broker{sessions: map[string]*liveSession{s.id: s}}
	live.ctx, live.cancel = context.WithCancel(bg)
	defer live.cancel()
	got := 0
	err := live.Subscribe(bg, s.id, 0, func(Event) error { got++; return nil })
	if got != 3 || err == nil {
		t.Fatalf("subscriber got %d events and %v", got, err)
	}
}

func TestCloseCommitsWhatIsStillPending(t *testing.T) {
	// Only Close can make these durable in time: commits wait an hour for company.
	b, _, _ := conversationBrokerOn(t, false, testStorageDelay(t, time.Hour))
	live := createConversation(t, b, "closing")
	session, _ := b.get(live.ID)
	for i := 0; i < 5; i++ {
		if err := b.receive(session, Event{Kind: "item.delta", TurnID: "", ItemID: "m", Role: "assistant", Text: "x"}); err != nil && !errors.Is(err, ErrStale) {
			t.Fatal(err)
		}
	}
	b.Close()
	session.mu.Lock()
	last := session.info.LastSeq
	session.mu.Unlock()
	if committed := committedEvents(t, b.store, live.ID); uint64(len(committed)) != last || last < 7 {
		t.Fatalf("%d of %d events are committed after Close", len(committed), last)
	}
}

func TestACrashLosesOnlyTheBoundedTailAndNeverAPromptThatWasSent(t *testing.T) {
	b, q, profile := queueBroker(t)
	s := createConversation(t, b, "crash")
	session, _ := b.get(s.ID)
	turn, err := b.Prompt(bg, s.ID, "sent-before-the-crash", PromptRequest{Text: "work"})
	if err != nil {
		t.Fatal(err)
	}
	nextQueued(t, q, "work")
	// The host dies: nothing is committed any more.
	c := counted(session)
	c.failure = errors.New("host is gone")
	for i := 0; i < 50; i++ {
		if err := b.receive(session, Event{Kind: "item.delta", TurnID: turn, ItemID: "m", Role: "assistant", Text: "lost"}); err != nil {
			t.Fatal(err)
		}
	}
	// A new host opens the same storage.
	restarted, err := NewBroker(testStorageFrom(t, b.store), []Profile{profile}, b.prepare, func(Profile, Sink, Ask) (Driver, error) {
		t.Error("restart launched a native client")
		return nil, errors.New("unexpected launch")
	})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	got, err := restarted.Get(bg, s.ID)
	if err != nil || got.State != "disconnected" {
		t.Fatalf("restored %+v: %v", got, err)
	}
	var journal []Event
	if err = restarted.Subscribe(withDeadline(t), s.ID, 0, func(e Event) error { journal = append(journal, e); return nil }); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	user, lost := 0, 0
	for i, e := range journal {
		if e.Seq != uint64(i+1) {
			t.Fatalf("journal has a gap at %d", i)
		}
		if e.Role == "user" && e.TurnID == turn {
			user++
		}
		if e.Text == "lost" {
			lost++
		}
	}
	if user != 1 || lost != 0 {
		t.Fatalf("restored journal has %d prompts and %d unflushed deltas", user, lost)
	}
	// The cursor of a client that had seen the lost tail is beyond the journal now.
	if err = restarted.Subscribe(bg, s.ID, uint64(len(journal)+50), func(Event) error { return nil }); !errors.Is(err, ErrCursor) {
		t.Fatalf("cursor beyond the journal: %v", err)
	}
	// And retrying the prompt with its key never sends it again.
	if again, err := restarted.Prompt(bg, s.ID, "sent-before-the-crash", PromptRequest{Text: "work"}); err != nil || again != turn {
		t.Fatalf("retry after the crash: %q %v", again, err)
	}
}

func withDeadline(t testing.TB) context.Context {
	ctx, cancel := context.WithTimeout(bg, 300*time.Millisecond)
	t.Cleanup(cancel)
	return ctx
}

// testStorageFrom opens the database of store again, as a new host would.
func testStorageFrom(t testing.TB, store *Store) *Store {
	t.Helper()
	inner := store.Client
	if c, ok := inner.(*countedStorage); ok {
		inner = c.StorageClient
	}
	other, err := NewStore(StorageBinding{Client: inner, Scope: store.Scope, DatabaseID: store.DatabaseID, Schema: store.Schema})
	if err != nil {
		t.Fatal(err)
	}
	return other
}

func TestANativeClientIsHeldBackWhileTooManyEventsWaitForTheirCommit(t *testing.T) {
	s := testSession(t, "s_pressure")
	c := counted(s)
	c.gate = make(chan struct{})
	b := &Broker{sessions: map[string]*liveSession{s.id: s}}
	b.ctx, b.cancel = context.WithCancel(bg)
	defer b.cancel()
	s.store.JournalDelay = time.Millisecond
	finished := make(chan int, 1)
	go func() {
		n := 0
		for ; n < maxPendingEvents+100; n++ {
			if err := b.receive(s, Event{Kind: "item.delta", TurnID: "", ItemID: "m", Role: "assistant", Text: "x"}); err != nil && !errors.Is(err, ErrStale) {
				break
			}
			s.mu.Lock()
			stale := s.current == ""
			s.mu.Unlock()
			if stale { // receive refuses events outside a turn: feed the journal directly
				s.mu.Lock()
				_ = s.appendLocked(Event{Kind: "item.delta", TurnID: "t_stream", ItemID: "m", Role: "assistant", Text: "x"})
				s.mu.Unlock()
				s.waitCapacity()
			}
		}
		finished <- n
	}()
	select {
	case <-finished:
		t.Fatal("the producer was not held back")
	case <-time.After(300 * time.Millisecond):
	}
	s.mu.Lock()
	pending := len(s.pending)
	s.mu.Unlock()
	if pending > maxPendingEvents+1 {
		t.Fatalf("%d events pending", pending)
	}
	close(c.gate)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("the producer was not released when commits resumed")
	}
	commit(t, s)
}

func TestARevisionMovedByAnotherHostIsRebuiltOnlyWhenTheConversationIsUntouched(t *testing.T) {
	a := testSession(t, "s_mine")
	counter := counted(a)
	otherStore, err := NewStore(a.store.StorageBinding)
	if err != nil {
		t.Fatal(err)
	}
	foreign := newSession(Session{SchemaVersion: Schema, ID: "s_theirs", Provider: "codex", Scope: scope("codex"), MetadataRevision: 1}, otherStore)
	foreign.profile = testProfile(t, "codex")
	if foreign.version, err = otherStore.create(bg, foreign.record()); err != nil {
		t.Fatal(err)
	}
	// The other host moved the scope revision with a commit to its own conversation.
	if err = foreign.appendLocked(Event{Kind: "notice", Text: "theirs"}); err != nil {
		t.Fatal(err)
	}
	commit(t, foreign)
	// This host's cached revision is stale: its next commit is rejected once, checked and rebuilt.
	before := counter.batches.Load()
	if err = a.appendLocked(Event{Kind: "notice", Text: "mine"}); err != nil {
		t.Fatal(err)
	}
	commit(t, a)
	if a.durable != 1 || counter.batches.Load()-before != 2 {
		t.Fatalf("durable=%d commits=%d", a.durable, counter.batches.Load()-before)
	}
	if events := stored(t, a, 0, 1); len(events) != 1 || events[0].Text != "mine" {
		t.Fatalf("events=%+v", events)
	}
}

func TestConcurrentCreatesOfOneKeyStoreOneConversation(t *testing.T) {
	b, _, _ := conversationBroker(t, false)
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := b.Create(bg, "same-key", scope("fixture"))
			if err != nil {
				t.Error(err)
				return
			}
			ids <- s.ID
		}()
	}
	wg.Wait()
	close(ids)
	first := ""
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Fatalf("one key produced %s and %s", first, id)
		}
	}
	rows, err := b.store.records(bg)
	if err != nil || len(rows) != 1 {
		t.Fatalf("stored conversations %d: %v", len(rows), err)
	}
}
