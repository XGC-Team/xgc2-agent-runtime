package agentruntime

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const MaxJournalBytes = 32 << 20
const MaxSessions = 128
const MaxEvents = 100000

var ErrCursor = errors.New("event cursor is ahead of this session")

type pendingInput struct {
	request Request
	answer  *Answer
	value   chan Answer
	active  bool
	cancel  context.CancelFunc
}
type liveSession struct {
	mu             sync.Mutex
	stopping       bool
	opening        context.CancelFunc
	info           Session
	events         []Event
	file           *os.File
	path           string
	runtimeClosing bool
	bytes          int64
	storageErr     error
	changed        chan struct{}
	driver         Driver
	cwd            string
	current        string
	cancel         context.CancelFunc
	ended          bool
	lastTurnStatus string
	queue          PromptQueue
	inputs         map[string]*pendingInput
	turns          map[string]string
	profile        Profile
}
type Broker struct {
	mu                sync.Mutex
	root              string
	profiles          map[string]Profile
	sessions          map[string]*liveSession
	prepare           Prepare
	factory           Factory
	closed            bool
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	settings          *settingsState
	decisionEvaluator DecisionEvaluator
}

func NewBroker(root string, profiles []Profile, prepare Prepare, factory Factory) (*Broker, error) {
	if prepare == nil {
		return nil, errors.New("workspace preparation is required")
	}
	if factory == nil {
		factory = NewDriver
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("agent runtime journal must be an owned private directory")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	b := &Broker{ctx: ctx, cancel: cancel, root: root, profiles: map[string]Profile{}, sessions: map[string]*liveSession{}, prepare: prepare, factory: factory}
	for _, p := range profiles {
		if _, ok := b.profiles[p.ID]; ok {
			return nil, errors.New("duplicate provider profile")
		}
		if _, err := commandArgs(p.Provider); err != nil {
			return nil, err
		}
		b.profiles[p.ID] = p
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		if err = b.load(filepath.Join(root, entry.Name())); err != nil {
			b.Close()
			return nil, err
		}
	}
	return b, nil
}

// Each log starts with a metadata record and contains only our normalized events.
// A torn trailing record is rejected, never silently truncated or treated as success.
func (b *Broker) load(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > MaxJournalBytes {
		return errors.New("invalid session journal")
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			file.Close()
		}
	}()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), MaxFrame)
	if !scanner.Scan() {
		return errors.New("journal metadata missing")
	}
	var record sessionRecord
	if json.Unmarshal(scanner.Bytes(), &record) != nil || record.SchemaVersion != Schema || !safeID.MatchString(record.ID) || record.Scope.Validate() != nil || filepath.Base(path) != record.ID+".jsonl" {
		return errors.New("invalid journal metadata")
	}
	infoRecord := record.Session
	if _, err := commandArgs(infoRecord.Provider); err != nil {
		return errors.New("invalid journal provider")
	}
	s := newSession(infoRecord, file)
	// Historical journals are validated as a stream, not retained in memory or
	// kept open. Only live conversations need an event cache.
	s.events = nil
	if record.ProfileSnapshot != nil {
		s.profile = *record.ProfileSnapshot
	} else {
		s.profile = b.profiles[infoRecord.Scope.ProfileID]
	}
	s.bytes = info.Size()
	for scanner.Scan() {
		var e Event
		if json.Unmarshal(scanner.Bytes(), &e) != nil || e.SchemaVersion != Schema || e.SessionID != infoRecord.ID || e.Provider != infoRecord.Provider || e.Seq != s.info.LastSeq+1 || s.info.LastSeq >= MaxEvents {
			return errors.New("journal event identity or sequence mismatch")
		}
		s.info.LastSeq = e.Seq
		if e.Kind == "session.identity" {
			s.info.AgentSessionID = e.AgentSessionID
		}
		if e.Kind == "session.state" {
			s.info.State = e.Status
		}
		s.applyMetadataLocked(e)
	}
	if scanner.Err() != nil {
		return errors.New("journal could not be read")
	}
	// Scanner accepts a final line without LF. Our writer never does: such a tail
	// is a crash boundary and cannot be safely appended to.
	if info.Size() > 0 {
		last := []byte{0}
		if _, err = file.ReadAt(last, info.Size()-1); err != nil || last[0] != '\n' {
			return errors.New("torn journal; retain for explicit recovery")
		}
	}
	if len(s.queue.Items) > 0 {
		s.pauseQueueLocked()
	}
	b.sessions[s.info.ID] = s
	ok = true
	if s.info.State != "closed" {
		if err = s.appendLocked(Event{Kind: "session.state", Status: "disconnected", Text: "Host restarted. Prior turn outcome may be unknown; prompts will not be resent."}); err != nil {
			return err
		}
	}
	s.releaseJournalLocked()
	return nil
}

type sessionRecord struct {
	Session
	ProfileSnapshot *Profile `json:"profileSnapshot,omitempty"`
}

func newSession(info Session, file *os.File) *liveSession {
	if info.MetadataRevision == 0 {
		info.MetadataRevision = 1
	}
	return &liveSession{info: info, file: file, path: file.Name(), events: []Event{}, changed: make(chan struct{}), inputs: map[string]*pendingInput{}, turns: map[string]string{}}
}
func (s *liveSession) appendLocked(e Event) error {
	if s.storageErr != nil {
		return s.storageErr
	}
	e.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	e.SchemaVersion = Schema
	e.SessionID = s.info.ID
	e.Provider = s.info.Provider
	e.Seq = s.info.LastSeq + 1
	if e.RuntimeID == "" && e.Kind != "session.metadata" {
		e.RuntimeID = s.info.RuntimeID
	}
	if e.TurnID == "" && (e.Kind != "session.state" && e.Kind != "session.identity") {
		e.TurnID = s.current
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if len(data)+1 >= MaxFrame || s.bytes+int64(len(data)+1) > MaxJournalBytes || s.info.LastSeq >= MaxEvents {
		err = errors.New("event journal capacity reached; retain and review this session")
	}
	if err == nil {
		file := s.file
		if file == nil {
			file, err = os.OpenFile(s.path, os.O_WRONLY|os.O_APPEND, 0600)
			if err == nil {
				defer file.Close()
			}
		}
		data = append(data, '\n')
		var n int
		if err == nil {
			n, err = file.Write(data)
		}
		if err == nil && n != len(data) {
			err = io.ErrShortWrite
		}
		if err == nil {
			err = file.Sync()
		}
	}
	if err != nil {
		s.storageErr = errors.New("event persistence failed; session stopped")
		s.info.State = "disconnected"
		close(s.changed)
		s.changed = make(chan struct{})
		return s.storageErr
	}
	s.bytes += int64(len(data))
	if s.events != nil {
		s.events = append(s.events, e)
	}
	s.info.LastSeq = e.Seq
	if e.Kind == "session.state" {
		s.info.State = e.Status
	}
	if e.Kind == "session.identity" {
		s.info.AgentSessionID = e.AgentSessionID
	}
	s.applyMetadataLocked(e)
	close(s.changed)
	s.changed = make(chan struct{})
	return nil
}
func (b *Broker) Providers() []Provider {
	_ = b.reloadSettingsProfiles()
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
	return result
}
func (b *Broker) Create(ctx context.Context, key string, c Create) (Session, error) {
	if err := c.Validate(); err != nil {
		return Session{}, err
	}
	if err := validKey(key); err != nil {
		return Session{}, err
	}
	replayID := "s_" + hash(c.Context.Kind + "\x00" + c.Context.ID + "\x00" + key)[:32]
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return Session{}, ErrUnavailable
	}
	existing := b.sessions[replayID]
	if existing != nil {
		existing.mu.Lock()
		info := existing.info
		existing.mu.Unlock()
		b.mu.Unlock()
		if info.Scope != c {
			return Session{}, ErrConflict
		}
		return info, nil
	}
	b.mu.Unlock()
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
	id := "s_" + hash(c.Context.Kind + "\x00" + c.Context.ID + "\x00" + key)[:32]
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return Session{}, ErrUnavailable
	}
	if s := b.sessions[id]; s != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.info.Scope != c {
			return Session{}, ErrConflict
		}
		return s.info, nil
	}
	p, ok := b.profiles[c.ProfileID]
	if !ok || !p.BillingReviewed || p.Disabled {
		return Session{}, ErrUnavailable
	}
	if err := checkExecutable(p); err != nil {
		return Session{}, err
	}
	selected, err := b.selection(p, c.Options)
	if err != nil {
		return Session{}, err
	}
	p.Defaults = selected
	if b.activeSessionsLocked() >= MaxSessions {
		return Session{}, errors.New("worker capacity reached; close a worker before starting another")
	}
	file, err := os.OpenFile(filepath.Join(b.root, id+".jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return Session{}, errors.New("cannot create the private session journal")
	}
	info := Session{SchemaVersion: Schema, ID: id, Scope: c, Provider: p.Provider, State: "starting", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Options: p.Defaults, MetadataRevision: 1, RuntimeID: "r_" + randomID()}
	data, _ := json.Marshal(sessionRecord{Session: info, ProfileSnapshot: &p})
	data = append(data, '\n')
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	if err != nil {
		file.Close()
		return Session{}, err
	}
	s := newSession(info, file)
	s.profile = p
	s.bytes = int64(len(data))
	b.sessions[id] = s
	b.wg.Add(1)
	go func() { defer b.wg.Done(); b.connect(ctx, s, p, true, info.RuntimeID) }()
	return info, nil
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
	defer func() { s.mu.Lock(); s.opening = nil; s.releaseJournalLocked(); s.mu.Unlock() }()
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
	s.mu.Unlock()
	if (err != nil || closed) && driver != nil {
		_ = driver.Close()
	}
}
func (b *Broker) receive(s *liveSession, e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopping || s.info.State == "closed" || s.info.Archived || (e.RuntimeID != "" && e.RuntimeID != s.info.RuntimeID) {
		return ErrStale
	}
	if e.Kind == "session.identity" {
		if e.AgentSessionID == "" || len(e.AgentSessionID) > 512 || (s.info.AgentSessionID != "" && s.info.AgentSessionID != e.AgentSessionID) {
			return errors.New("session identity changed")
		}
	} else if e.TurnID != "" && e.TurnID != s.current {
		return ErrStale
	}
	if e.Kind == "turn.end" {
		if s.current == "" || s.ended {
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
		return err
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
func (b *Broker) Get(id string) (Session, error) {
	s, err := b.get(id)
	if err != nil {
		return Session{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info, nil
}
func (b *Broker) List() []Session {
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
func (b *Broker) Prompt(id, key, prompt string) (string, error) {
	return b.PromptWithOptions(id, key, prompt, AgentOptions{})
}
func (b *Broker) PromptWithOptions(id, key, prompt string, options AgentOptions) (string, error) {
	if err := validKey(key); err != nil {
		return "", err
	}
	if strings.TrimSpace(prompt) == "" || len(prompt) > 128<<10 || !utf8.ValidString(prompt) {
		return "", errors.New("prompt must be nonempty UTF-8 up to 128 KiB")
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return "", ErrUnavailable
	}
	candidate := b.sessions[id]
	if candidate == nil {
		b.mu.Unlock()
		return "", ErrNotFound
	}
	candidate.mu.Lock()
	preTurn := "t_" + hash(id + "\x00" + key)[:32]
	fingerprintBefore := promptFingerprint(prompt, options)
	previous, replay, lookupErr := candidate.findTurnLocked(preTurn)
	if lookupErr != nil {
		candidate.mu.Unlock()
		b.mu.Unlock()
		return "", lookupErr
	}
	if replay {
		candidate.mu.Unlock()
		b.mu.Unlock()
		if previous != fingerprintBefore {
			return "", ErrConflict
		}
		return preTurn, nil
	}
	if candidate.info.State != "ready" || candidate.driver == nil || candidate.current != "" {
		candidate.mu.Unlock()
		b.mu.Unlock()
		return "", ErrUnavailable
	}
	profile := candidate.profile
	candidate.mu.Unlock()
	b.mu.Unlock()
	b.warmSelection(b.ctx, profile)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return "", ErrUnavailable
	}
	s := b.sessions[id]
	if s == nil {
		return "", ErrNotFound
	}
	var err error
	s.mu.Lock()
	defer s.mu.Unlock()
	turn := "t_" + hash(id + "\x00" + key)[:32]
	selected, err := b.selection(s.profile, options)
	if err != nil {
		return "", err
	}
	fingerprint := promptFingerprint(prompt, options)
	if existing, ok := s.turns[turn]; ok {
		if existing != fingerprint {
			return "", ErrConflict
		}
		return turn, nil
	}
	if s.info.State != "ready" || s.driver == nil || s.current != "" {
		return "", ErrUnavailable
	}
	return b.startPromptLocked(s, turn, prompt, options, selected)
}

// Caller owns both broker and session locks.
func (b *Broker) startPromptLocked(s *liveSession, turn, prompt string, options, selected AgentOptions) (string, error) {
	s.current = turn
	s.ended = false
	s.lastTurnStatus = ""
	if err := s.appendLocked(Event{Kind: "item.snapshot", TurnID: turn, ItemID: "user", Role: "user", Text: prompt, Status: "submitted", Details: promptDetails(options)}); err != nil {
		return "", err
	}
	s.turns[turn] = promptFingerprint(prompt, options)
	if err := s.appendLocked(Event{Kind: "session.state", Status: "running", TurnID: turn}); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(b.ctx, 4*time.Hour)
	s.cancel = cancel
	driver := s.driver
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
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
		if err != nil || !ended || s.lastTurnStatus != "completed" {
			s.pauseQueueLocked()
		}
		s.current = ""
		s.cancel = nil
		if !s.stopping && s.info.State != "closed" {
			state := "ready"
			if err != nil || !ended {
				state = "disconnected"
			}
			_ = s.appendLocked(Event{Kind: "session.state", Status: state})
		}
		s.releaseJournalLocked()
		s.mu.Unlock()
		cancel()
		if err != nil || !ended {
			_ = driver.Close()
		}
		b.drainQueue(s.info.ID)
	}()
	return turn, nil
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
		return Answer{}, errors.New("too many pending user requests")
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
	remaining := false
	for _, p := range s.inputs {
		if p.active {
			remaining = true
		}
	}
	if !remaining && s.current == turn && !s.ended && s.info.State == "awaiting-input" {
		_ = s.appendLocked(Event{Kind: "session.state", Status: "running", TurnID: turn})
	}
	s.releaseJournalLocked()
	s.mu.Unlock()
	return answer, err
}
func (b *Broker) Answer(id, requestID string, a Answer) error {
	return b.AnswerContext(context.Background(), id, requestID, a)
}
func (b *Broker) AnswerContext(ctx context.Context, id, requestID string, a Answer) error {
	s, err := b.get(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.inputs[requestID]
	if p == nil {
		return ErrStale
	}
	if p.answer != nil {
		old, _ := json.Marshal(p.answer)
		next, _ := json.Marshal(a)
		if string(old) == string(next) {
			return nil
		}
		return ErrConflict
	}
	if s.stopping || !p.active || s.ended || s.info.State == "closed" || s.info.State == "cancelling" {
		return ErrStale
	}
	if err = p.request.ValidateAnswer(a); err != nil {
		return err
	}
	// The journal records that a response was submitted, not that the provider
	// granted permission or the tool succeeded. Never persist free-form answers.
	if err = s.appendLocked(Event{Kind: "input.submitted", ItemID: requestID, Status: "submitted", Decision: decisionReceipt(ctx, p.request, a)}); err != nil {
		return err
	}
	p.answer = &a
	p.value <- a
	return nil
}
func (b *Broker) Cancel(ctx context.Context, id string) error {
	s, err := b.get(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.stopping || s.current == "" || s.driver == nil {
		s.mu.Unlock()
		return ErrStale
	}
	driver := s.driver
	for _, p := range s.inputs {
		if p.active {
			p.cancel()
		}
	}
	err = s.appendLocked(Event{Kind: "session.state", Status: "cancelling"})
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return driver.Cancel(ctx) // Only native acknowledgement/exit can finish the turn.
}
func (b *Broker) Reconnect(id string) error { return b.ReconnectContext(context.Background(), id) }

// ReconnectContext carries ephemeral composition metadata into explicit resume.
// It is not persisted and cannot change the durable session scope.
func (b *Broker) ReconnectContext(ctx context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return ErrUnavailable
	}
	if b.activeSessionsLocked() >= MaxSessions {
		return ErrUnavailable
	}
	s := b.sessions[id]
	if s == nil {
		return ErrNotFound
	}
	var err error
	s.mu.Lock()
	if (s.info.State != "disconnected" && s.info.State != "closed") || s.current != "" || s.storageErr != nil || s.opening != nil || s.runtimeClosing || s.info.Archived {
		s.mu.Unlock()
		return ErrConflict
	}
	p := s.profile
	if p.ID == "" || p.Provider != s.info.Provider {
		s.mu.Unlock()
		return ErrUnavailable
	}
	if err = s.activateJournalLocked(); err != nil {
		s.mu.Unlock()
		return err
	}
	runtimeID := "r_" + randomID()
	err = s.appendLocked(Event{Kind: "session.runtime", RuntimeID: runtimeID})
	if err == nil {
		err = s.appendLocked(Event{Kind: "session.state", Status: "starting", Text: "Resumed the existing session. No prior prompt is replayed."})
	}
	s.mu.Unlock()
	if err == nil {
		b.wg.Add(1)
		go func() { defer b.wg.Done(); b.connect(ctx, s, p, false, runtimeID) }()
	}
	return err
}
func (b *Broker) CloseSession(id string) error {
	s, err := b.get(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	finish, err := s.retireRuntimeLocked()
	s.mu.Unlock()
	finish()
	return err
}

// Replay is bounded per response, cursor-checked and scoped to one journal.
// Browsers reconnect to this log, never to Prompt.
func (b *Broker) Replay(id string, after uint64) ([]Event, <-chan struct{}, error) {
	s, err := b.get(id)
	if err != nil {
		return nil, nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if after > s.info.LastSeq {
		return nil, nil, ErrCursor
	}
	end := min(s.info.LastSeq, after+256)
	var result []Event
	if s.events != nil {
		result = append([]Event{}, s.events[after:end]...)
	} else {
		result, err = s.readEventsLocked(after, end)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(result) == 0 && s.storageErr != nil {
		return nil, nil, s.storageErr
	}
	return result, s.changed, nil
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
		if s.info.State != "closed" {
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
	// Wait for all opens and active turn finalizers before closing the journals.
	b.wg.Wait()
	for _, s := range sessions {
		s.mu.Lock()
		if s.file != nil {
			_ = s.file.Close()
			s.file = nil
		}
		s.storageErr = ErrUnavailable
		s.mu.Unlock()
	}
	return nil
}
func validKey(key string) error {
	if !safeID.MatchString(key) {
		return errors.New("Idempotency-Key must be 1-96 safe ASCII characters")
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
