package agentruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/XGC-Team/xgc2-storage/api"
)

type historyReadCounter struct {
	StorageClient
	calls      atomic.Int64
	eventCalls atomic.Int64
	eventBytes atomic.Int64
}

func (c *historyReadCounter) Snapshot(ctx context.Context, id string, req api.SnapshotRequest) (api.SnapshotResponse, error) {
	c.calls.Add(1)
	for _, q := range req.Queries {
		if q.Collection == "events" {
			c.eventCalls.Add(1)
		}
	}
	out, err := c.StorageClient.Snapshot(ctx, id, req)
	if err == nil {
		for _, result := range out.Results {
			if result.Collection == "events" {
				for _, row := range result.Records {
					c.eventBytes.Add(int64(len(row.Data)))
				}
			}
		}
	}
	return out, err
}

func seedHistory(t *testing.T, store *Store, profile Profile, id string, count, textSize int) *liveSession {
	t.Helper()
	s := newSession(Session{SchemaVersion: Schema, ID: id, Provider: profile.Provider, Scope: scope(profile.ID), State: "closed", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), MetadataRevision: 1}, store)
	s.profile = profile
	var err error
	s.version, err = store.create(context.Background(), s.record())
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("x", textSize)
	for i := 0; i < count; i++ {
		s.mu.Lock()
		err := s.appendLocked(Event{Kind: "notice", ItemID: fmt.Sprint(i), Text: text})
		s.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if textSize > 500000 {
			commit(t, s)
		}
	}
	commit(t, s)
	return s
}

func TestThinCatalogAttentionAndSelectedHistoryReads(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	for i := 0; i < 220; i++ {
		history := seedHistory(t, store, profile, fmt.Sprintf("s_hist_%03d", i), 1, 48<<10)
		history.mu.Lock()
		err := history.appendLocked(Event{Kind: "prompt.queue", Queue: &PromptQueue{Revision: 1, Items: []QueuedPrompt{}}})
		history.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		commit(t, history)
	}
	counter := &historyReadCounter{StorageClient: store.Client}
	store.Client = counter
	factory := func(_ Profile, sink Sink, ask Ask) (Driver, error) {
		return &conversationDriver{sink: sink, ask: ask}, nil
	}
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return t.TempDir(), nil }, factory)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if counter.eventCalls.Load() != 0 {
		t.Fatalf("startup read %d transcript pages", counter.eventCalls.Load())
	}
	active, err := b.Create(context.Background(), "selected-active", scope("codex"))
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, b, active.ID, "ready")
	if _, err = b.Prompt(context.Background(), active.ID, "turn", PromptRequest{Text: "active"}); err != nil {
		t.Fatal(err)
	}
	request := waitRequest(t, b, active.ID)
	live, err := b.get(active.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = live.flushAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	counter.calls.Store(0)
	counter.eventCalls.Store(0)
	counter.eventBytes.Store(0)
	seen := map[string]bool{}
	cursor := ""
	for {
		page, err := b.List(context.Background(), SessionListOptions{Limit: 17, After: cursor, Context: scope("codex").Context})
		if err != nil {
			t.Fatal(err)
		}
		for _, session := range page.Sessions {
			if seen[session.ID] {
				t.Fatalf("duplicate %s", session.ID)
			}
			seen[session.ID] = true
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	attention, err := b.Attention(context.Background())
	if err != nil || len(attention.Sessions) != 1 || len(attention.Sessions[0].Pending) != 1 || len(seen) != 221 || counter.calls.Load() != 0 {
		t.Fatalf("catalog=%d attention=%+v calls=%d err=%v", len(seen), attention, counter.calls.Load(), err)
	}
	if current, err := b.Current(context.Background(), active.ID); err != nil || len(current.Inputs) != 1 || counter.calls.Load() != 0 {
		t.Fatalf("current=%+v reads=%d err=%v", current, counter.calls.Load(), err)
	}
	if current, err := b.Current(context.Background(), "s_hist_000"); err != nil || current.Queue == nil || current.Queue.Revision != 1 || len(current.Queue.Items) != 0 || counter.calls.Load() != 0 {
		t.Fatalf("empty retained queue=%+v reads=%d err=%v", current.Queue, counter.calls.Load(), err)
	}
	page, err := b.History(context.Background(), "s_hist_000", HistoryOptions{})
	if err != nil || len(page.Events) != 2 || page.Events[0].Seq != 1 || page.Events[1].Seq != 2 || page.NextBefore != 0 {
		t.Fatalf("selected history: %+v %v", page, err)
	}
	if counter.eventCalls.Load() != 1 || counter.eventBytes.Load() < 48<<10 || counter.eventBytes.Load() > 49<<10 {
		t.Fatalf("selected read: calls=%d bytes=%d", counter.eventCalls.Load(), counter.eventBytes.Load())
	}
	if _, err = b.History(context.Background(), "s_hist_000", HistoryOptions{Before: 4}); !errors.Is(err, ErrCursor) {
		t.Fatalf("out-of-range cursor: %v", err)
	}
	if _, err = b.List(context.Background(), SessionListOptions{After: strings.Repeat("x", 2000)}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversized catalog cursor: %v", err)
	}
	if err = b.Answer(context.Background(), active.ID, request.ID, Answer{OptionID: "yes"}); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, active.ID, "ready")
}

func TestHistorySuffixAndDetachedReplayRespectStorageByteBounds(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	seedHistory(t, store, profile, "s_large", 6, 1<<20)
	counter := &historyReadCounter{StorageClient: store.Client}
	store.Client = counter
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	page, err := b.History(context.Background(), "s_large", HistoryOptions{Limit: 6})
	if err != nil || len(page.Events) == 0 || page.Events[len(page.Events)-1].Seq != 6 || page.NextBefore != page.Events[0].Seq || counter.eventBytes.Load() > replayBytes {
		t.Fatalf("suffix=%+v readBytes=%d err=%v", page, counter.eventBytes.Load(), err)
	}
	counter.eventBytes.Store(0)
	first, _, err := b.replay("s_large", 0)
	if err != nil || len(first) == 0 || first[0].Seq != 1 || counter.eventBytes.Load() > replayBytes {
		t.Fatalf("replay len=%d readBytes=%d err=%v", len(first), counter.eventBytes.Load(), err)
	}
	before := page.NextBefore
	page, err = b.History(context.Background(), "s_large", HistoryOptions{Before: before, Limit: 6})
	if err != nil || len(page.Events) == 0 || page.Events[len(page.Events)-1].Seq != before-1 {
		t.Fatalf("previous page: %+v %v", page, err)
	}
}

func TestCleanupProtectsPendingTurnAndRemovesPromptAndHistory(t *testing.T) {
	b, _, _ := conversationBroker(t, false)
	s := createConversation(t, b, "protected")
	turn, err := b.Prompt(context.Background(), s.ID, "turn", PromptRequest{Text: "protected prompt"})
	if err != nil {
		t.Fatal(err)
	}
	request := waitRequest(t, b, s.ID)
	if _, err = b.CleanupHistory(context.Background(), s.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("pending turn was not protected: %v", err)
	}
	if err = b.Answer(context.Background(), s.ID, request.ID, Answer{OptionID: "yes"}); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, s.ID, "ready")
	if err = b.CloseSession(context.Background(), s.ID); err != nil {
		t.Fatal(err)
	}
	_, release, err := b.retainReader(s.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.CleanupHistory(context.Background(), s.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("reader not protected: %v", err)
	}
	release()
	before := b.historyBytes.Load()
	result, err := b.CleanupHistory(context.Background(), s.ID)
	if err != nil || !result.Complete || result.DeletedEvents == 0 || b.historyBytes.Load() >= before {
		t.Fatalf("cleanup=%+v bytes %d -> %d err=%v", result, before, b.historyBytes.Load(), err)
	}
	if _, err = b.Get(context.Background(), s.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("session survived: %v", err)
	}
	read, err := b.store.snapshot(context.Background(), []api.Query{{Collection: "prompts", Keys: []string{turn}}}, nil)
	if err != nil || !read.Results[0].Records[0].Deleted {
		t.Fatalf("prompt identity not retired: %+v %v", read, err)
	}
	b.Close()
	reopened, err := NewBroker(b.store, nil, b.prepare, b.factory)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err = reopened.Get(context.Background(), s.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("reopened session: %v", err)
	}
}

func TestPartialCleanupResumesAfterReopen(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	seedHistory(t, store, profile, "s_partial", 300, 20)
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := b.CleanupHistory(context.Background(), "s_partial")
	if err != nil || result.Complete || result.DeletedEvents != retentionStepEvents*retentionStepsPerCall {
		t.Fatalf("partial cleanup=%+v %v", result, err)
	}
	if _, err = b.History(context.Background(), "s_partial", HistoryOptions{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("partial transcript visible: %v", err)
	}
	if _, err = b.Get(context.Background(), "s_partial"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("partially retired session visible: %v", err)
	}
	if gotScope, err := b.CleanupScope(context.Background(), "s_partial"); err != nil || gotScope != scope(profile.ID).Context {
		t.Fatalf("cleanup authorization scope: %+v %v", gotScope, err)
	}
	b.Close()
	reopened, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	result, err = reopened.CleanupHistory(context.Background(), "s_partial")
	if err != nil && !errors.Is(err, ErrNotFound) {
		t.Fatalf("resume=%+v %v", result, err)
	}
	if err == nil && !result.Complete {
		t.Fatalf("resume incomplete: %+v", result)
	}
	if _, err = reopened.Get(context.Background(), "s_partial"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cleanup not durable: %v", err)
	}
}

func TestJournalAdmissionIncludesInflightBatch(t *testing.T) {
	s := testSession(t, "s_buffer")
	counter := counted(s)
	counter.gate = make(chan struct{})
	text := strings.Repeat("x", 1<<20)
	var failed error
	for i := 0; i < 8; i++ {
		s.mu.Lock()
		failed = s.appendLocked(Event{Kind: "notice", Text: text})
		bytes, count := s.pendingBytes, len(s.pending)
		s.mu.Unlock()
		if bytes > maxBufferedBytes || count > maxBufferedEvents {
			t.Fatalf("unbounded queue: %d/%d", bytes, count)
		}
		if failed != nil {
			break
		}
	}
	if !errors.Is(failed, ErrCapacity) {
		t.Fatalf("unreported capacity: %v", failed)
	}
	close(counter.gate)
	if err := s.flushAll(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestStalledSubscribersAreBoundedAndReleaseOnCancellation(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	seedHistory(t, store, profile, "s_readers", 4, 1024)
	counter := &historyReadCounter{StorageClient: store.Client}
	store.Client = counter
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	s, err := b.get("s_readers")
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	unobserved := s.changed
	if err = s.appendLocked(Event{Kind: "notice", Text: "no subscribers"}); err != nil {
		s.mu.Unlock()
		t.Fatal(err)
	}
	if s.changed != unobserved {
		s.mu.Unlock()
		t.Fatal("unobserved event notified a subscriber")
	}
	s.mu.Unlock()
	commit(t, s)
	counter.eventCalls.Store(0)
	arrived := make(chan struct{}, maxReadersPerSession)
	unblock := make(chan struct{})
	done := make(chan error, maxReadersPerSession)
	cancel := make([]context.CancelFunc, 0, maxReadersPerSession)
	for i := 0; i < maxReadersPerSession; i++ {
		ctx, stop := context.WithCancel(context.Background())
		cancel = append(cancel, stop)
		go func() {
			done <- b.Subscribe(ctx, "s_readers", 0, func(Event) error { arrived <- struct{}{}; <-unblock; return nil })
		}()
	}
	for i := 0; i < maxReadersPerSession; i++ {
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			t.Fatal("subscriber did not reach sink")
		}
	}
	if counter.eventCalls.Load() != maxReadersPerSession {
		t.Fatalf("%d readers used %d event reads", maxReadersPerSession, counter.eventCalls.Load())
	}
	if err = b.Subscribe(context.Background(), "s_readers", 0, func(Event) error { return nil }); !errors.Is(err, ErrCapacity) {
		t.Fatalf("extra reader: %v", err)
	}
	if _, err = b.CleanupHistory(context.Background(), "s_readers"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stalled readers not protected: %v", err)
	}
	for _, stop := range cancel {
		stop()
	}
	close(unblock)
	for i := 0; i < maxReadersPerSession; i++ {
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("subscriber: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("subscriber did not join")
		}
	}
	s.mu.Lock()
	readers, subscribers := s.readers, s.subscribers
	s.mu.Unlock()
	if readers != 0 || subscribers != 0 {
		t.Fatalf("readers=%d subscribers=%d", readers, subscribers)
	}
}

func TestRetentionOnStartupOnlyForExpiredUnsettledFreeHistory(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	s := newSession(Session{SchemaVersion: Schema, ID: "s_old", Provider: profile.Provider, Scope: scope(profile.ID), State: "closed", CreatedAt: time.Now().Add(-HistoryMaxAge - time.Hour).UTC().Format(time.RFC3339Nano), MetadataRevision: 1}, store)
	s.profile = profile
	var err error
	s.version, err = store.create(context.Background(), s.record())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.appendLocked(Event{Kind: "notice", Text: "expired"}); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.updatedAt = s.info.CreatedAt
	s.pending[0].record.UpdatedAt = s.info.CreatedAt
	s.mu.Unlock()
	commit(t, s)
	seedHistory(t, store, profile, "s_recent", 1, 32)
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	b.wg.Wait()
	if _, err = b.Get(context.Background(), "s_old"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired history remains: %v", err)
	}
	if page, err := b.History(context.Background(), "s_recent", HistoryOptions{}); err != nil || len(page.Events) != 1 {
		t.Fatalf("recent history lost: %+v %v", page, err)
	}
}

func TestCurrentSeedKeepsActiveRequestAndCompletedTurnAtExactCursor(t *testing.T) {
	b, _, _ := conversationBroker(t, false)
	session := createConversation(t, b, "seed")
	turn, err := b.Prompt(context.Background(), session.ID, "seed-prompt", PromptRequest{Text: "accepted prompt"})
	if err != nil {
		t.Fatal(err)
	}
	request := waitRequest(t, b, session.ID)
	live, err := b.get(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = b.receive(live, Event{Kind: "item.snapshot", TurnID: turn, ItemID: "tool", Role: "tool", Text: "tool output", Details: map[string]any{"type": "commandExecution", "command": "inspect"}}); err != nil {
		t.Fatal(err)
	}
	if err = live.flushAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	counter := &historyReadCounter{StorageClient: b.store.Client}
	b.store.Client = counter
	seed, err := b.Current(context.Background(), session.ID)
	if err != nil || seed.Session.LastSeq == 0 || seed.ActiveTurnID != turn || len(seed.Items) != 2 || seed.Items[0].Text != "accepted prompt" || seed.Items[1].Text != "tool output" || len(seed.Inputs) != 1 || seed.Inputs[0].Request.ID != request.ID || counter.calls.Load() != 0 {
		t.Fatalf("active seed=%+v reads=%d err=%v", seed, counter.calls.Load(), err)
	}
	cursor := seed.Session.LastSeq
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	received := make(chan Event, 20)
	done := make(chan error, 1)
	go func() {
		done <- b.Subscribe(ctx, session.ID, cursor, func(e Event) error { received <- e; return nil })
	}()
	if err = b.Answer(context.Background(), session.ID, request.ID, Answer{OptionID: "yes"}); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, session.ID, "ready")
	seed, err = b.Current(context.Background(), session.ID)
	if err != nil || seed.ActiveTurnID != "" || seed.Result == nil || seed.Result.TurnID != turn || seed.Result.Status != "completed" || len(seed.Items) != 2 || len(seed.Inputs) != 0 {
		t.Fatalf("completed seed=%+v err=%v", seed, err)
	}
	select {
	case first := <-received:
		if first.Seq != cursor+1 {
			t.Fatalf("cursor jumped: %d -> %d", cursor, first.Seq)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("missing suffix")
	}
	cancel()
	if err = <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("subscriber: %v", err)
	}
	if err = b.CloseSession(context.Background(), session.ID); err != nil {
		t.Fatal(err)
	}
	if err = b.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err := NewBroker(b.store, nil, b.prepare, b.factory)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	seed, err = restored.Current(context.Background(), session.ID)
	if err != nil || len(seed.Items) != 0 || seed.HistoryBefore != seed.Session.LastSeq+1 {
		t.Fatalf("restored history demand=%+v %v", seed, err)
	}
}

func TestCurrentSeedCompactsDeltasAndFailsHonestlyAtBound(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	s := newSession(Session{SchemaVersion: Schema, ID: "s_seed", Provider: "codex", Scope: scope("codex"), State: "ready", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), MetadataRevision: 1}, store)
	s.profile = profile
	var err error
	s.version, err = store.create(context.Background(), s.record())
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	s = b.sessions[s.id]
	s.mu.Lock()
	s.current = "t_active"
	if err = s.appendLocked(Event{Kind: "item.snapshot", TurnID: "t_active", ItemID: "user", Role: "user", Status: "submitted", Text: "question"}); err != nil {
		s.mu.Unlock()
		t.Fatal(err)
	}
	for i := 0; i < 769; i++ {
		if err = s.appendLocked(Event{Kind: "item.delta", TurnID: "t_active", ItemID: "reply", Role: "assistant", Text: "x"}); err != nil {
			s.mu.Unlock()
			t.Fatal(err)
		}
	}
	s.mu.Unlock()
	seed, err := b.Current(context.Background(), s.id)
	if err != nil || len(seed.Items) != 2 || seed.Items[1].Text != strings.Repeat("x", 769) || seed.Session.LastSeq != 770 {
		t.Fatalf("compacted seed=%+v %v", seed, err)
	}
	s.mu.Lock()
	if err = s.appendLocked(Event{Kind: "item.delta", TurnID: "t_active", ItemID: "reply", Role: "assistant", Text: strings.Repeat("z", maxCurrentText)}); err != nil {
		s.mu.Unlock()
		t.Fatal(err)
	}
	s.mu.Unlock()
	if _, err = b.Current(context.Background(), s.id); !errors.Is(err, ErrCapacity) {
		t.Fatalf("oversized seed returned silently: %v", err)
	}
}

func TestCurrentSeedFencesTheSubscriptionHandoff(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	seedHistory(t, store, profile, "s_handoff", 1, 16)
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	seed, err := b.Current(context.Background(), "s_handoff")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.CleanupHistory(context.Background(), "s_handoff"); !errors.Is(err, ErrConflict) {
		t.Fatalf("seed handoff was retired: %v", err)
	}
	_, release, err := b.retainReader("s_handoff", true)
	if err != nil {
		t.Fatal(err)
	}
	release()
	result, err := b.CleanupHistory(context.Background(), "s_handoff")
	if err != nil || !result.Complete || seed.Session.LastSeq != 1 {
		t.Fatalf("handoff cleanup=%+v seed=%+v err=%v", result, seed, err)
	}
}

type blockingCloseDriver struct {
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (*blockingCloseDriver) Open(context.Context, string, string) error   { return nil }
func (*blockingCloseDriver) Prompt(context.Context, string, string) error { return nil }
func (*blockingCloseDriver) Cancel(context.Context, string) error         { return nil }
func (d *blockingCloseDriver) Close() error {
	if d.calls.Add(1) == 1 {
		close(d.entered)
	}
	<-d.release
	return nil
}

func TestCleanupCancellationJoinsOneInFlightDriverClose(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	seedHistory(t, store, profile, "s_close_wait", 1, 16)
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	driver := &blockingCloseDriver{entered: make(chan struct{}), release: make(chan struct{})}
	s := b.sessions["s_close_wait"]
	s.mu.Lock()
	s.driver = driver
	s.info.State = "ready"
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = b.CleanupHistory(ctx, s.id)
	if !errors.Is(err, context.DeadlineExceeded) {
		close(driver.release)
		t.Fatalf("cleanup cancellation: %v", err)
	}
	select {
	case <-driver.entered:
	default:
		close(driver.release)
		t.Fatal("driver close not started")
	}
	rows, err := store.records(context.Background())
	if err != nil || len(rows) != 1 || rows[0].Retiring || rows[0].LastSeq != 1 {
		close(driver.release)
		t.Fatalf("unjoined close retired data: %+v %v", rows, err)
	}
	close(driver.release)
	result, err := b.CleanupHistory(context.Background(), s.id)
	if err != nil || !result.Complete || driver.calls.Load() != 1 {
		t.Fatalf("joined cleanup=%+v calls=%d err=%v", result, driver.calls.Load(), err)
	}
}

func TestCleanupRejectsConcurrentCallersWithoutQueueAndResumesAfterCancellation(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	seedHistory(t, store, profile, "s_cleanup_busy", 40, 16)
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	counter := &historyReadCounter{StorageClient: store.Client}
	store.Client = counter
	driver := &blockingCloseDriver{entered: make(chan struct{}), release: make(chan struct{})}
	released := false
	defer func() {
		if !released {
			close(driver.release)
		}
	}()
	s := b.sessions["s_cleanup_busy"]
	s.mu.Lock()
	s.driver = driver
	s.info.State = "ready"
	s.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := make(chan error, 1)
	go func() { _, err := b.CleanupHistory(ctx, s.id); first <- err }()
	select {
	case <-driver.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("driver close did not start")
	}

	assertRejected := func() {
		t.Helper()
		const callers = 256
		contenders, stop := context.WithCancel(context.Background())
		defer stop()
		start, done := make(chan struct{}), make(chan error, callers)
		for i := 0; i < callers; i++ {
			go func() {
				<-start
				_, err := b.CleanupHistory(contenders, s.id)
				done <- err
			}()
		}
		close(start)
		deadline := time.NewTimer(3 * time.Second)
		defer deadline.Stop()
		for i := 0; i < callers; i++ {
			select {
			case err := <-done:
				if !errors.Is(err, ErrCapacity) || errorCode(err) != "resource_exhausted" {
					t.Fatalf("contender %d returned %v", i, err)
				}
			case <-deadline.C:
				t.Fatalf("cleanup callers queued behind the blocked close: %d/%d returned", i, callers)
			}
		}
		if counter.calls.Load() != 0 || driver.calls.Load() != 1 {
			t.Fatalf("rejected callers did work: storage reads=%d driver closes=%d", counter.calls.Load(), driver.calls.Load())
		}
	}
	assertRejected()
	response := serve(Handler(b, HandlerOptions{}), "POST", "/sessions/"+s.id+"/cleanup-history", "", nil)
	if code, _, _ := response.error(t); response.status != 429 || code != "resource_exhausted" {
		t.Fatalf("cleanup HTTP refusal: status=%d code=%s", response.status, code)
	}
	canceled, stop := context.WithCancel(context.Background())
	stop()
	if _, err = b.CleanupHistory(canceled, s.id); !errors.Is(err, context.Canceled) {
		t.Fatalf("already canceled caller: %v", err)
	}

	cancel()
	select {
	case err = <-first:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled cleanup: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled cleanup did not release admission")
	}
	type completion struct {
		result HistoryCleanup
		err    error
	}
	continued := make(chan completion, 1)
	go func() {
		result, err := b.CleanupHistory(context.Background(), s.id)
		continued <- completion{result, err}
	}()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for len(b.retentionSlot) == 0 {
		select {
		case <-deadline.C:
			t.Fatal("continuation did not join the in-flight close")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	assertRejected()
	if driver.calls.Load() != 1 {
		t.Fatalf("driver closed %d times", driver.calls.Load())
	}
	close(driver.release)
	released = true
	select {
	case outcome := <-continued:
		if outcome.err != nil || !outcome.result.Complete || outcome.result.DeletedEvents != 40 || driver.calls.Load() != 1 {
			t.Fatalf("cleanup continuation=%+v err=%v driver closes=%d", outcome.result, outcome.err, driver.calls.Load())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup did not finish after joining the driver")
	}
	if _, err = b.Get(context.Background(), s.id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("retired session remains: %v", err)
	}
	if counter.calls.Load() != 3 {
		t.Fatalf("continuation used %d storage reads for 40 events; want 3 bounded steps", counter.calls.Load())
	}
}

func TestBrokerCloseCancelsAdmittedCleanupAndJoinsItsDriver(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	seedHistory(t, store, profile, "s_cleanup_shutdown", 1, 16)
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	driver := &blockingCloseDriver{entered: make(chan struct{}), release: make(chan struct{})}
	released := false
	defer func() {
		if !released {
			close(driver.release)
		}
		_ = b.Close()
	}()
	s := b.sessions["s_cleanup_shutdown"]
	s.mu.Lock()
	s.driver = driver
	s.info.State = "ready"
	s.mu.Unlock()
	cleanup := make(chan error, 1)
	go func() { _, err := b.CleanupHistory(context.Background(), s.id); cleanup <- err }()
	select {
	case <-driver.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("driver close did not start")
	}
	closed := make(chan error, 1)
	go func() { closed <- b.Close() }()
	select {
	case err = <-cleanup:
		if !errors.Is(err, ErrUnavailable) {
			t.Fatalf("shutdown cleanup: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown left the admitted cleanup waiting")
	}
	select {
	case err = <-closed:
		t.Fatalf("broker did not join the driver close: %v", err)
	default:
	}
	close(driver.release)
	released = true
	select {
	case err = <-closed:
		if err != nil || driver.calls.Load() != 1 {
			t.Fatalf("broker shutdown: %v, driver closes=%d", err, driver.calls.Load())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("broker did not join after driver release")
	}
	rows, err := store.records(context.Background())
	if err != nil || len(rows) != 1 || rows[0].Retiring || rows[0].LastSeq != 1 {
		t.Fatalf("cancelled shutdown retired data: %+v %v", rows, err)
	}
	reopened, err := NewBroker(store, nil, b.prepare, b.factory)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	result, err := reopened.CleanupHistory(context.Background(), s.id)
	if err != nil || !result.Complete || result.DeletedEvents != 1 {
		t.Fatalf("cleanup after reopen: %+v %v", result, err)
	}
}

func TestAutomaticRetentionQueuesOnlyOneWorkerBehindCleanup(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	seedHistory(t, store, profile, "s_cleanup_auto", 1, 16)
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	counter := &historyReadCounter{StorageClient: store.Client}
	store.Client = counter
	driver := &blockingCloseDriver{entered: make(chan struct{}), release: make(chan struct{})}
	released := false
	defer func() {
		if !released {
			close(driver.release)
		}
	}()
	s := b.sessions["s_cleanup_auto"]
	s.mu.Lock()
	s.driver = driver
	s.info.State = "ready"
	s.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := make(chan error, 1)
	go func() { _, err := b.CleanupHistory(ctx, s.id); first <- err }()
	select {
	case <-driver.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("driver close did not start")
	}
	var triggers sync.WaitGroup
	for i := 0; i < 128; i++ {
		triggers.Add(1)
		go func() { defer triggers.Done(); b.scheduleRetention() }()
	}
	triggered := make(chan struct{})
	go func() { triggers.Wait(); close(triggered) }()
	select {
	case <-triggered:
	case <-time.After(2 * time.Second):
		t.Fatal("automatic retention triggers did not return")
	}
	b.mu.Lock()
	running := b.retentionRunning
	b.mu.Unlock()
	if !running || len(b.retentionSlot) != 1 || driver.calls.Load() != 1 || counter.calls.Load() != 0 {
		t.Fatalf("automatic worker state: running=%v slot=%d closes=%d reads=%d", running, len(b.retentionSlot), driver.calls.Load(), counter.calls.Load())
	}
	if _, err = b.CleanupHistory(context.Background(), s.id); !errors.Is(err, ErrCapacity) {
		t.Fatalf("manual caller was queued: %v", err)
	}
	cancel()
	select {
	case err = <-first:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("initial cleanup: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("initial cleanup did not cancel")
	}
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for len(b.retentionSlot) == 0 {
		select {
		case <-deadline.C:
			t.Fatal("automatic worker did not join the in-flight close")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if _, err = b.CleanupHistory(context.Background(), s.id); !errors.Is(err, ErrCapacity) {
		t.Fatalf("manual caller queued behind the automatic worker: %v", err)
	}
	close(driver.release)
	released = true
	joined := make(chan struct{})
	go func() { b.wg.Wait(); close(joined) }()
	select {
	case <-joined:
	case <-time.After(2 * time.Second):
		t.Fatal("automatic cleanup did not join")
	}
	if _, err = b.Get(context.Background(), s.id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("automatic continuation did not retire the session: %v", err)
	}
	if counter.calls.Load() != 1 || driver.calls.Load() != 1 {
		t.Fatalf("automatic cleanup work: reads=%d closes=%d", counter.calls.Load(), driver.calls.Load())
	}
}

func TestCurrentSeedBoundsAllPendingRequestBytes(t *testing.T) {
	b, _, _ := conversationBroker(t, false)
	session := createConversation(t, b, "many-pending")
	if _, err := b.Prompt(context.Background(), session.ID, "turn", PromptRequest{Text: "start"}); err != nil {
		t.Fatal(err)
	}
	original := waitRequest(t, b, session.ID)
	s, err := b.get(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{}, 9)
	for i := 0; i < 9; i++ {
		go func() {
			_, _ = b.ask(s, ctx, Request{Kind: "permission", Title: "large pending request", Text: strings.Repeat("x", 240<<10), Options: []Option{{ID: "yes", Kind: "allow_once"}}})
			done <- struct{}{}
		}()
	}
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		pending, err := b.Inputs(context.Background(), session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(pending) == 10 {
			break
		}
		select {
		case <-deadline.C:
			t.Fatalf("only %d requests became active", len(pending))
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if _, err = b.Current(context.Background(), session.ID); !errors.Is(err, ErrCapacity) {
		t.Fatalf("unbounded current input payload: %v", err)
	}
	cancel()
	for i := 0; i < 9; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("pending request did not join")
		}
	}
	if err = b.Answer(context.Background(), session.ID, original.ID, Answer{OptionID: "yes"}); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, session.ID, "ready")
}

func TestClosedBrokerRejectsCleanupEvenIfSlotIsFree(t *testing.T) {
	store := testStorage(t)
	profile := testProfile(t, "codex")
	seedHistory(t, store, profile, "s_closed_broker", 1, 8)
	b, err := NewBroker(store, []Profile{profile}, func(context.Context, Create, string, bool) (string, error) { return "", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = b.Close(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err = b.CleanupHistory(context.Background(), "s_closed_broker"); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("closed cleanup attempt %d: %v", i, err)
		}
	}
	rows, err := store.records(context.Background())
	if err != nil || len(rows) != 1 || rows[0].Retiring {
		t.Fatalf("closed broker changed history: %+v %v", rows, err)
	}
}
