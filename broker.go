package agentruntime

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	xrpc "github.com/XGC-Team/xgc2-xrpc/go"
)

const MaxSessions = 128
const MaxEvents = 100000

// replayPage is how many events one storage read returns to a subscriber.
const replayPage = 256

type pendingInput struct {
	request   Request
	answer    *Answer
	answerSeq uint64
	value     chan Answer
	active    bool
	cancel    context.CancelFunc
}

// turnRecord is what a live conversation remembers of a prompt it accepted: the
// fingerprint that makes a replay with other content a conflict, and the
// sequence number of its user message, which a replay waits to be durable.
type turnRecord struct {
	fingerprint string
	seq         uint64
}
type liveSession struct {
	mu                  sync.Mutex
	id                  string
	stopping            bool
	opening             context.CancelFunc
	info                Session
	store               *Store
	queueSeq            uint64
	submittedQueueTurns []string
	runtimeClosing      bool
	bytes               int64
	storageErr          error
	changed             chan struct{}
	driver              Driver
	cwd                 string
	current             string
	stopped             string
	cancel              context.CancelFunc
	ended               bool
	lastTurnStatus      string
	queue               PromptQueue
	inputs              map[string]*pendingInput
	turns               map[string]turnRecord
	profile             Profile
	// The journal (journal.go): events numbered but not committed yet, the
	// highest committed sequence number and the version of the stored record.
	pending      []pendingEvent
	pendingBytes int64
	durable      uint64
	version      string
	flushing     bool
	flushNow     bool
	persistErr   error
	wake         chan struct{}
	flushed      chan struct{}
}
type Broker struct {
	mu                sync.Mutex
	store             *Store
	profiles          map[string]Profile
	sessions          map[string]*liveSession
	creating          map[string]chan struct{}
	prepare           Prepare
	factory           Factory
	closed            bool
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	settings          *settingsState
	decisionEvaluator DecisionEvaluator
}

func NewBroker(store *Store, profiles []Profile, prepare Prepare, factory Factory, files ...*NativeFiles) (*Broker, error) {
	if len(files) > 1 {
		return nil, errors.New("one host native file capability required")
	}
	if store == nil || prepare == nil {
		return nil, errors.New("storage and workspace preparation are required")
	}
	if factory == nil {
		factory = NewDriver
	}
	ctx, cancel := context.WithCancel(context.Background())
	if len(files) == 1 && files[0] != nil {
		ctx = WithNativeFiles(ctx, files[0])
	}
	b := &Broker{ctx: ctx, cancel: cancel, store: store, profiles: map[string]Profile{}, sessions: map[string]*liveSession{}, creating: map[string]chan struct{}{}, prepare: prepare, factory: factory}
	for _, p := range profiles {
		if _, ok := b.profiles[p.ID]; ok {
			cancel()
			return nil, errors.New("duplicate provider profile")
		}
		if _, err := commandArgs(p.Provider); err != nil {
			cancel()
			return nil, err
		}
		b.profiles[p.ID] = p
	}
	records, err := store.records(context.Background())
	if err != nil {
		cancel()
		return nil, err
	}
	for _, loaded := range records {
		record := loaded.sessionRecord
		s := newSession(record.Session, store)
		s.durable, s.version = record.LastSeq, loaded.version
		s.profile = record.ProfileSnapshot
		s.bytes = record.Bytes
		s.queueSeq = record.QueueSeq
		s.submittedQueueTurns = record.SubmittedQueueTurns
		s.lastTurnStatus = record.LastTurnStatus
		if record.QueueSeq > 0 {
			events, err := store.readEvents(record.ID, record.QueueSeq-1, record.QueueSeq)
			if err != nil || events[0].Queue == nil {
				cancel()
				return nil, errors.Join(errors.New("persisted queue unavailable"), err)
			}
			s.queue = cloneQueue(*events[0].Queue)
			for _, turn := range record.SubmittedQueueTurns {
				s.applyQueueEventLocked(Event{Kind: "item.snapshot", Role: "user", Status: "submitted", TurnID: turn})
			}
		}
		b.sessions[s.info.ID] = s
		if len(s.queue.Items) > 0 {
			// Startup restores the current host projection without requiring more
			// durable capacity. The next explicit queue command records its change.
			s.queue.Paused = true
		}
		if s.info.State != "closed" && s.info.State != "disconnected" {
			// No worker exists in this host until explicit reconnect. Preserve the
			// recorded outcome and history; a full session cannot block all history.
			s.info.State = "disconnected"
		}
	}
	return b, nil
}

func newSession(info Session, store *Store) *liveSession {
	if info.MetadataRevision == 0 {
		info.MetadataRevision = 1
	}
	return &liveSession{id: info.ID, info: info, store: store, changed: make(chan struct{}), flushed: make(chan struct{}), wake: make(chan struct{}, 1), inputs: map[string]*pendingInput{}, turns: map[string]turnRecord{}}
}

func (s *liveSession) record() sessionRecord {
	return sessionRecord{Session: s.info, ProfileSnapshot: s.profile, Bytes: s.bytes, QueueSeq: s.queueSeq, SubmittedQueueTurns: append([]string{}, s.submittedQueueTurns...), LastTurnStatus: s.lastTurnStatus}
}

func (b *Broker) Providers(ctx context.Context) ([]Provider, error) {
	if err := b.reloadSettingsProfiles(); err != nil {
		return nil, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	result := []Provider{}
	for _, p := range b.profiles {
		protocol := "acp/v1"
		toolMode := "native"
		interactive := true
		if p.Provider == "codex" {
			protocol = "codex/app-server"
		}
		if p.Provider == "claude" {
			protocol = "claude/stream-json"
			interactive = b.settings != nil
			if !interactive {
				toolMode = "Read,Glob,Grep only"
			}
		}
		err := checkExecutable(p)
		detail := "Operator-reviewed binary. Login and billing stay with the CLI. This is not a live subscription check."
		if err != nil {
			detail = err.Error()
		}
		result = append(result, Provider{ID: p.ID, Provider: p.Provider, Protocol: protocol, Available: err == nil && p.BillingReviewed && !p.Disabled, Detail: detail, ReviewedVersion: p.ReviewedVersion, InteractiveRequests: interactive, ToolMode: toolMode})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (b *Broker) Create(ctx context.Context, key string, c Create) (Session, error) {
	if err := c.Validate(); err != nil {
		return Session{}, err
	}
	if err := validKey(key); err != nil {
		return Session{}, err
	}
	id := "s_" + hash(c.Context.Kind + "\x00" + c.Context.ID + "\x00" + key)[:32]
	if info, found, err := b.createdSession(ctx, id, c); found || err != nil {
		return info, err
	}
	if err := b.reloadSettingsProfiles(); err != nil {
		return Session{}, err
	}
	b.mu.Lock()
	candidate, available := b.profiles[c.ProfileID]
	b.mu.Unlock()
	if !available || !candidate.BillingReviewed || candidate.Disabled {
		return Session{}, ErrUnavailable
	}
	b.warmSelection(ctx, candidate)
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return Session{}, ErrUnavailable
	}
	if b.sessions[id] != nil || b.creating[id] != nil {
		b.mu.Unlock()
		info, _, err := b.createdSession(ctx, id, c)
		return info, err
	}
	p, ok := b.profiles[c.ProfileID]
	if !ok || !p.BillingReviewed || p.Disabled {
		b.mu.Unlock()
		return Session{}, ErrUnavailable
	}
	if err := checkExecutable(p); err != nil {
		b.mu.Unlock()
		return Session{}, err
	}
	selected, err := b.selection(p, c.Options)
	if err != nil {
		b.mu.Unlock()
		return Session{}, err
	}
	p.Defaults = selected
	if b.activeSessionsLocked() >= MaxSessions {
		b.mu.Unlock()
		return Session{}, exhausted("worker capacity reached; close a worker before starting another")
	}
	if len(b.sessions)+len(b.creating) >= MaxRetainedSessions {
		b.mu.Unlock()
		return Session{}, exhausted("retained conversation capacity reached")
	}
	info := Session{SchemaVersion: Schema, ID: id, Scope: c, Provider: p.Provider, State: "starting", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Options: p.Defaults, MetadataRevision: 1, RuntimeID: "r_" + randomID()}
	s := newSession(info, b.store)
	s.profile = p
	// The record is written without the broker lock; a repeated create waits
	// here for its outcome.
	created := make(chan struct{})
	b.creating[id] = created
	b.mu.Unlock()
	version, err := b.store.create(ctx, s.record())
	b.mu.Lock()
	delete(b.creating, id)
	close(created)
	if err != nil {
		b.mu.Unlock()
		return Session{}, err
	}
	if b.closed {
		b.mu.Unlock()
		return Session{}, ErrUnavailable
	}
	s.version = version
	b.sessions[id] = s
	b.wg.Add(1)
	b.mu.Unlock()
	go func() { defer b.wg.Done(); b.connect(ctx, s, p, true, info.RuntimeID) }()
	return info, nil
}

// createdSession answers a repeated create from the conversation it made, once
// that one is stored.
func (b *Broker) createdSession(ctx context.Context, id string, c Create) (Session, bool, error) {
	for {
		b.mu.Lock()
		if b.closed {
			b.mu.Unlock()
			return Session{}, true, ErrUnavailable
		}
		if created := b.creating[id]; created != nil {
			b.mu.Unlock()
			select {
			case <-created:
				continue
			case <-ctx.Done():
				return Session{}, true, ctx.Err()
			}
		}
		existing := b.sessions[id]
		b.mu.Unlock()
		if existing == nil {
			return Session{}, false, nil
		}
		existing.mu.Lock()
		info := existing.info
		existing.mu.Unlock()
		if info.Scope != c {
			return Session{}, true, ErrConflict
		}
		return info, true, nil
	}
}
func (b *Broker) connect(binding context.Context, s *liveSession, p Profile, fresh bool, runtimeID string) {
	ctx, cancel := context.WithTimeout(sessionBindingContext{Context: b.ctx, values: binding}, 65*time.Second)
	defer cancel()
	s.mu.Lock()
	scope, id, native := s.info.Scope, s.info.ID, s.info.AgentSessionID
	if s.stopping || s.info.State == "closed" || s.info.Archived || s.info.RuntimeID != runtimeID {
		s.mu.Unlock()
		return
	}
	s.opening = cancel
	old := s.driver
	s.driver = nil
	s.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	attempted := false
	defer func() {
		if !attempted {
			s.mu.Lock()
			s.opening = nil
			s.releaseHistoryLocked()
			s.mu.Unlock()
		}
	}()
	cwd, err := b.prepare(ctx, scope, id, fresh)
	if err == nil {
		err = ctx.Err()
	}
	var driver Driver
	if err == nil {
		driver, err = b.factory(p, func(e Event) error { e.RuntimeID = runtimeID; return b.receive(s, e) }, func(ctx context.Context, r Request) (Answer, error) { return b.askRuntime(s, ctx, r, runtimeID) })
	}
	if err == nil {
		if scoped, ok := driver.(SessionScopedDriver); ok {
			err = scoped.BindSession(ctx, scope, id)
		}
	}
	if err == nil {
		s.mu.Lock()
		if s.stopping || s.info.State == "closed" || s.info.Archived || ctx.Err() != nil {
			s.mu.Unlock()
			_ = driver.Close()
			return
		}
		s.driver = driver
		s.cwd = cwd
		s.mu.Unlock()
		err = driver.Open(ctx, cwd, native)
	}
	s.mu.Lock()
	if !s.stopping && s.info.State != "closed" && err != nil {
		_ = s.appendLocked(Event{Kind: "notice", Status: "error", Text: "Agent connection failed. Check the pinned CLI, sign-in, and the approved workspace. No provider fallback occurred."})
		_ = s.appendLocked(Event{Kind: "session.state", Status: "disconnected"})
	} else if !s.stopping && s.info.State != "closed" {
		err = s.appendLocked(Event{Kind: "session.state", Status: "ready"})
	}
	closed := s.stopping || s.info.State == "closed"
	// The attempt is over before its outcome is visible: a caller that sees the
	// session ready can reconnect, close or prompt it at once, and a newer attempt
	// is never mistaken for this one.
	attempted = true
	s.opening = nil
	s.releaseHistoryLocked()
	s.mu.Unlock()
	if (err != nil || closed) && driver != nil {
		_ = driver.Close()
	}
}
func (b *Broker) receive(s *liveSession, e Event) error {
	s.waitCapacity()
	s.mu.Lock()
	if s.stopping || s.info.State == "closed" || s.info.Archived || (e.RuntimeID != "" && e.RuntimeID != s.info.RuntimeID) {
		s.mu.Unlock()
		return ErrStale
	}
	if e.Kind == "session.identity" {
		if e.AgentSessionID == "" || len(e.AgentSessionID) > 512 || (s.info.AgentSessionID != "" && s.info.AgentSessionID != e.AgentSessionID) {
			s.mu.Unlock()
			return errors.New("session identity changed")
		}
	} else if e.TurnID != "" && e.TurnID != s.current {
		s.mu.Unlock()
		return ErrStale
	}
	if e.Kind == "turn.end" {
		if s.current == "" || s.ended {
			s.mu.Unlock()
			return ErrStale
		}
		s.ended = true
		for _, p := range s.inputs {
			if p.active {
				p.cancel()
			}
		}
	}
	if err := s.appendLocked(e); err != nil {
		s.mu.Unlock()
		return err
	}
	seq := s.info.LastSeq
	s.mu.Unlock()
	if durableKind(e) {
		// The identity a resume needs and the end of a turn are on disk before
		// the native client is told they were taken.
		return s.waitDurable(context.Background(), seq)
	}
	return nil
}
func (b *Broker) get(id string) (*liveSession, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.sessions[id]
	if s == nil {
		return nil, ErrNotFound
	}
	return s, nil
}
func (b *Broker) Get(_ context.Context, id string) (Session, error) {
	s, err := b.get(id)
	if err != nil {
		return Session{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info, nil
}
func (b *Broker) list() []Session {
	b.mu.Lock()
	sessions := make([]*liveSession, 0, len(b.sessions))
	for _, s := range b.sessions {
		sessions = append(sessions, s)
	}
	b.mu.Unlock()
	result := []Session{}
	for _, s := range sessions {
		s.mu.Lock()
		result = append(result, s.info)
		s.mu.Unlock()
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt == result[j].CreatedAt {
			return result[i].ID > result[j].ID
		}
		return result[i].CreatedAt > result[j].CreatedAt
	})
	return result
}
func (b *Broker) Prompt(ctx context.Context, id, key string, request PromptRequest) (string, error) {
	prompt, options := request.Text, request.Options
	if err := validKey(key); err != nil {
		return "", err
	}
	if !validQueuedText(prompt) {
		return "", invalid("prompt must be nonempty UTF-8 up to 128 KiB")
	}
	s, err := b.open(id)
	if err != nil {
		return "", err
	}
	turn := turnIdentity(id, key)
	fingerprint := promptFingerprint(prompt, options)
	// A replay is answered before anything else is checked, and never sends again.
	if replayed, err := s.replayedTurn(turn, fingerprint); replayed || err != nil {
		if err != nil {
			return "", err
		}
		return turn, nil
	}
	s.mu.Lock()
	profile := s.profile
	known, accepted := s.turns[turn] // accepted while this request looked at storage
	ready := s.info.State == "ready" && s.driver != nil && s.current == ""
	s.mu.Unlock()
	if accepted {
		if known.fingerprint != fingerprint {
			return "", ErrConflict
		}
		return turn, s.waitDurable(context.Background(), known.seq)
	}
	if !ready {
		return "", ErrUnavailable
	}
	b.warmSelection(ctx, profile)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return "", ErrUnavailable
	}
	s.mu.Lock()
	if known, ok := s.turns[turn]; ok { // an identical request was admitted while this one inspected the provider
		s.mu.Unlock()
		b.mu.Unlock()
		if known.fingerprint != fingerprint {
			return "", ErrConflict
		}
		return turn, s.waitDurable(context.Background(), known.seq)
	}
	selected, err := b.selection(s.profile, options)
	if err == nil && (s.info.State != "ready" || s.driver == nil || s.current != "") {
		err = ErrUnavailable
	}
	var admitted <-chan error
	if err == nil {
		admitted, err = b.startPromptLocked(s, turn, prompt, options, selected)
	}
	s.mu.Unlock()
	b.mu.Unlock()
	if err != nil {
		return "", err
	}
	if err = <-admitted; err != nil {
		return "", err
	}
	return turn, nil
}

// open returns the conversation id of a broker that is still running.
func (b *Broker) open(id string) (*liveSession, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, ErrUnavailable
	}
	s := b.sessions[id]
	if s == nil {
		return nil, ErrNotFound
	}
	return s, nil
}

func turnIdentity(id, key string) string { return "t_" + hash(id + "\x00" + key)[:32] }

// startPromptLocked accepts the turn: its user message and the running state
// are appended, and a worker goroutine sends the prompt to the native client
// once the message is durable. The returned channel reports that moment, or the
// failure that ended the turn before it started. Caller owns both broker and
// session locks.
func (b *Broker) startPromptLocked(s *liveSession, turn, prompt string, options, selected AgentOptions) (<-chan error, error) {
	s.current = turn
	s.ended = false
	s.lastTurnStatus = ""
	fail := func(err error) (<-chan error, error) {
		s.current = ""
		return nil, err
	}
	if err := s.appendLocked(Event{Kind: "item.snapshot", TurnID: turn, ItemID: "user", Role: "user", Text: prompt, Status: "submitted", Details: promptDetails(options)}); err != nil {
		return fail(err)
	}
	seq := s.info.LastSeq
	s.turns[turn] = turnRecord{promptFingerprint(prompt, options), seq}
	if err := s.appendLocked(Event{Kind: "session.state", Status: "running", TurnID: turn}); err != nil {
		return fail(err)
	}
	ctx, cancel := context.WithTimeout(b.ctx, 4*time.Hour)
	s.cancel = cancel
	admitted := make(chan error, 1)
	b.wg.Add(1)
	go b.runTurn(ctx, cancel, s, s.driver, turn, prompt, selected, seq, admitted)
	return admitted, nil
}

// runTurn is the worker of one turn. The prompt reaches the native client only
// after its user message is committed, so a crash can lose a prompt that was
// never sent but can never send one that was lost.
func (b *Broker) runTurn(ctx context.Context, cancel context.CancelFunc, s *liveSession, driver Driver, turn, prompt string, selected AgentOptions, seq uint64, admitted chan<- error) {
	defer b.wg.Done()
	if err := s.waitDurable(context.Background(), seq); err != nil {
		s.mu.Lock()
		if s.current == turn {
			s.current, s.stopped, s.cancel = "", "", nil
		}
		s.mu.Unlock()
		cancel()
		admitted <- err
		return
	}
	admitted <- nil
	var err error
	if configurable, ok := driver.(optionDriver); ok {
		err = configurable.PromptWithOptions(ctx, turn, prompt, selected)
	} else if selected != (AgentOptions{}) {
		err = errors.New("driver does not support selected options")
	} else {
		err = driver.Prompt(ctx, turn, prompt)
	}
	s.mu.Lock()
	ended := s.ended
	if !ended {
		_ = s.appendLocked(Event{Kind: "turn.end", TurnID: turn, Status: "unknown", Text: "The client transport ended without a confirmed terminal result. This prompt will not be resent."})
	}
	if err != nil {
		_ = s.appendLocked(Event{Kind: "notice", TurnID: turn, Status: "error", Text: "The turn failed or disconnected. Check the client for authentication, quota, or runtime issues."})
	}
	for _, p := range s.inputs {
		if p.active {
			p.cancel()
		}
	}
	if err != nil || !ended || s.lastTurnStatus != "completed" || s.stopped == turn {
		s.pauseQueueLocked()
	}
	s.current = ""
	s.stopped = ""
	s.cancel = nil
	if !s.stopping && s.info.State != "closed" {
		state := "ready"
		if err != nil || !ended {
			state = "disconnected"
		}
		_ = s.appendLocked(Event{Kind: "session.state", Status: state})
	}
	s.releaseHistoryLocked()
	s.mu.Unlock()
	cancel()
	if err != nil || !ended {
		_ = driver.Close()
	}
	b.drainQueue(s.id)
}
func (b *Broker) ask(s *liveSession, ctx context.Context, r Request) (Answer, error) {
	return b.askRuntime(s, ctx, r, "")
}
func (b *Broker) askRuntime(s *liveSession, ctx context.Context, r Request, runtimeID string) (Answer, error) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return Answer{}, ErrUnavailable
	}
	b.wg.Add(1)
	b.mu.Unlock()
	defer b.wg.Done()
	if err := validateRequest(r); err != nil {
		return Answer{}, err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.mu.Lock()
	if s.stopping || s.current == "" || s.ended || s.info.State == "closed" || s.info.State == "cancelling" || s.info.Archived || (runtimeID != "" && runtimeID != s.info.RuntimeID) {
		s.mu.Unlock()
		return Answer{}, ErrStale
	}
	active := 0
	for _, p := range s.inputs {
		if p.active {
			active++
		}
	}
	if active >= 16 {
		s.mu.Unlock()
		return Answer{}, exhausted("too many pending user requests")
	}
	r.ID = "q_" + randomID()
	r.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	p := &pendingInput{request: r, value: make(chan Answer, 1), active: true, cancel: cancel}
	s.inputs[r.ID] = p
	turn := s.current
	err := s.appendLocked(Event{Kind: "input.request", TurnID: turn, ItemID: r.ID, Request: &r})
	if err == nil {
		err = s.appendLocked(Event{Kind: "session.state", Status: "awaiting-input", TurnID: turn})
	}
	info, runtimeCwd := s.info, s.cwd
	s.mu.Unlock()
	if err != nil {
		return Answer{}, err
	}
	b.evaluatePending(ctx, info, r, runtimeCwd)
	var answer Answer
	select {
	case answer = <-p.value:
	case <-ctx.Done():
		err = ctx.Err()
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	s.mu.Lock()
	p.active = false
	status := "answered"
	if err != nil {
		status = "expired"
		answer = Answer{Cancel: true}
	}
	_ = s.appendLocked(Event{Kind: "input.resolved", TurnID: turn, ItemID: r.ID, Status: status})
	seq := s.info.LastSeq
	remaining := false
	for _, p := range s.inputs {
		if p.active {
			remaining = true
		}
	}
	if !remaining && s.current == turn && !s.ended && s.info.State == "awaiting-input" {
		_ = s.appendLocked(Event{Kind: "session.state", Status: "running", TurnID: turn})
	}
	s.releaseHistoryLocked()
	s.mu.Unlock()
	// The native client learns the answer only once its resolution is on disk.
	if werr := s.waitDurable(context.Background(), seq); werr != nil {
		return Answer{Cancel: true}, werr
	}
	return answer, err
}
func (b *Broker) Answer(ctx context.Context, id, requestID string, a Answer) error {
	s, err := b.get(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	p := s.inputs[requestID]
	if p == nil {
		s.mu.Unlock()
		return ErrStale
	}
	if p.answer != nil {
		old, _ := json.Marshal(p.answer)
		next, _ := json.Marshal(a)
		seq := p.answerSeq
		s.mu.Unlock()
		if string(old) != string(next) {
			return ErrConflict
		}
		return s.waitDurable(context.Background(), seq)
	}
	if s.stopping || !p.active || s.ended || s.info.State == "closed" || s.info.State == "cancelling" {
		s.mu.Unlock()
		return ErrStale
	}
	if err = p.request.ValidateAnswer(a); err != nil {
		s.mu.Unlock()
		return err
	}
	// The journal records that a response was submitted, not that the provider
	// granted permission or the tool succeeded. Never persist free-form answers.
	if err = s.appendLocked(Event{Kind: "input.submitted", ItemID: requestID, Status: "submitted", Decision: decisionReceipt(ctx, p.request, a)}); err != nil {
		s.mu.Unlock()
		return err
	}
	p.answer = &a
	p.answerSeq = s.info.LastSeq
	s.mu.Unlock()
	// The receipt is durable before the native client sees the answer.
	if err = s.waitDurable(context.Background(), p.answerSeq); err != nil {
		return err
	}
	p.value <- a
	return nil
}

// Cancel is the operator's Stop. It holds the queued messages first: a stop is
// a decision about what runs next, and the held queue stays held when the turn
// ends however it ends. The turn enters cancelling only after the native client
// accepted the stop, so a stop that failed leaves its approvals answerable and
// the turn running; only native acknowledgement or exit finishes the turn.
func (b *Broker) Cancel(ctx context.Context, id string) error {
	s, err := b.get(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.stopping || s.current == "" || s.ended || s.driver == nil {
		s.mu.Unlock()
		return ErrStale
	}
	turn, driver := s.current, s.driver
	s.stopped = turn
	s.pauseQueueLocked()
	s.mu.Unlock()
	if err = driver.Cancel(ctx, turn); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != turn || s.ended || s.stopping || s.info.State == "cancelling" {
		return nil // the turn ended on its own while the stop was delivered
	}
	for _, p := range s.inputs {
		if p.active {
			p.cancel()
		}
	}
	return s.appendLocked(Event{Kind: "session.state", Status: "cancelling"})
}

// Reconnect carries the ephemeral composition values of ctx into the explicit
// resume. They are not persisted and cannot change the durable session scope.
func (b *Broker) Reconnect(ctx context.Context, id string) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrUnavailable
	}
	if b.activeSessionsLocked() >= MaxSessions {
		b.mu.Unlock()
		return ErrUnavailable
	}
	s := b.sessions[id]
	if s == nil {
		b.mu.Unlock()
		return ErrNotFound
	}
	s.mu.Lock()
	if (s.info.State != "disconnected" && s.info.State != "closed") || s.current != "" || s.storageErr != nil || s.opening != nil || s.runtimeClosing || s.info.Archived {
		s.mu.Unlock()
		b.mu.Unlock()
		return ErrConflict
	}
	p := s.profile
	if p.ID == "" || p.Provider != s.info.Provider {
		s.mu.Unlock()
		b.mu.Unlock()
		return ErrUnavailable
	}
	runtimeID := "r_" + randomID()
	err := s.appendLocked(Event{Kind: "session.runtime", RuntimeID: runtimeID})
	if err == nil {
		err = s.appendLocked(Event{Kind: "session.state", Status: "starting", Text: "Resumed the existing session. No prior prompt is replayed."})
	}
	seq := s.info.LastSeq
	s.mu.Unlock()
	b.mu.Unlock()
	if err != nil {
		return err
	}
	// The new runtime is on disk before a worker starts under it, so a second
	// host resuming the same conversation is turned away here, not later.
	if err = s.waitDurable(context.Background(), seq); err != nil {
		return err
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrUnavailable
	}
	b.wg.Add(1)
	b.mu.Unlock()
	go func() { defer b.wg.Done(); b.connect(ctx, s, p, false, runtimeID) }()
	return nil
}
func (b *Broker) CloseSession(_ context.Context, id string) error {
	s, err := b.get(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	finish, err := s.retireRuntimeLocked()
	seq := s.info.LastSeq
	s.mu.Unlock()
	finish()
	if err != nil {
		return err
	}
	return s.waitDurable(context.Background(), seq)
}

// replay is bounded per response, cursor-checked and scoped to one conversation.
// Subscribers reconnect to this log, never to Prompt. It reads what is committed
// from storage, without any lock, and the rest from the journal's pending list.
func (b *Broker) replay(id string, after uint64) ([]Event, <-chan struct{}, error) {
	s, err := b.get(id)
	if err != nil {
		return nil, nil, err
	}
	s.mu.Lock()
	if after > s.info.LastSeq {
		s.mu.Unlock()
		return nil, nil, ErrCursor
	}
	end := min(s.info.LastSeq, after+replayPage)
	tail, durable := s.tailLocked(after, end)
	changed, failure := s.changed, s.storageErr
	s.mu.Unlock()
	var result []Event
	if after < durable {
		if result, err = s.store.readEvents(id, after, min(end, durable)); err != nil {
			return nil, nil, err
		}
		for _, event := range result {
			if event.Provider != s.info.Provider {
				return nil, nil, ErrConflict
			}
		}
	}
	result = append(result, tail...)
	if len(result) == 0 && failure != nil {
		return nil, nil, failure
	}
	return result, changed, nil
}
func (b *Broker) Subscribe(ctx context.Context, id string, after uint64, emit func(Event) error) error {
	for {
		events, changed, err := b.replay(id, after)
		if err != nil {
			return err
		}
		for _, e := range events {
			if err = emit(e); err != nil {
				return err
			}
			after = e.Seq
		}
		if len(events) > 0 {
			continue // more may have arrived, or the conversation may have failed, meanwhile
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-b.ctx.Done():
			return ErrUnavailable
		case <-changed:
		}
	}
}
func (b *Broker) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.cancel()
	sessions := make([]*liveSession, 0, len(b.sessions))
	for _, s := range b.sessions {
		sessions = append(sessions, s)
	}
	b.mu.Unlock()
	for _, s := range sessions {
		s.mu.Lock()
		s.stopping = true
		driver, cancel := s.driver, s.cancel
		for _, p := range s.inputs {
			if p.active {
				p.cancel()
			}
		}
		if s.info.State != "closed" && s.info.State != "disconnected" {
			_ = s.appendLocked(Event{Kind: "session.state", Status: "disconnected", Text: "Local host stopped; no automatic task retry."})
		}
		s.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		if driver != nil {
			_ = driver.Close()
		}
	}
	// Wait for all opens and active turn finalizers before retiring the host.
	b.wg.Wait()
	// Everything the workers appended is committed before the storage may go.
	ctx, stop := context.WithTimeout(context.Background(), closeFlushTimeout)
	defer stop()
	var flushing sync.WaitGroup
	for _, s := range sessions {
		flushing.Add(1)
		go func() { defer flushing.Done(); _ = s.flushAll(ctx) }()
	}
	flushing.Wait()
	for _, s := range sessions {
		s.mu.Lock()
		s.storageErr = ErrUnavailable
		s.mu.Unlock()
	}
	return nil
}

// closeFlushTimeout bounds how long Close waits for the last commits.
const closeFlushTimeout = 15 * time.Second

// validKey accepts the XRPC request id grammar, which is what the XRPC service
// passes as the idempotency key.
func validKey(key string) error {
	if !xrpc.ValidID(key) {
		return invalid("the idempotency key must match [A-Za-z0-9._:-]{1,128}")
	}
	return nil
}
func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func randomID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("OS random source failed: %v", err))
	}
	return hex.EncodeToString(b[:])
}
