package agentruntime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Session.ID is the durable conversation identity. Metadata changes and worker
// restarts append to its existing conversation; they never create another transcript.
type SessionMetadata struct {
	Title            string `json:"title"`
	Archived         bool   `json:"archived"`
	MetadataRevision uint64 `json:"metadataRevision"`
}

type MetadataUpdate struct {
	ExpectedRevision uint64  `json:"expectedRevision"`
	Title            *string `json:"title,omitempty"`
	Archived         *bool   `json:"archived,omitempty"`
}

func (b *Broker) UpdateMetadata(_ context.Context, id string, update MetadataUpdate) (Session, error) {
	if update.ExpectedRevision == 0 || (update.Title == nil && update.Archived == nil) {
		return Session{}, invalid("metadata revision and an explicit change are required")
	}
	if update.Title != nil {
		title := *update.Title
		if !utf8.ValidString(title) || strings.TrimSpace(title) != title || utf8.RuneCountInString(title) > 160 || strings.ContainsFunc(title, unicode.IsControl) {
			return Session{}, invalid("conversation title must be at most 160 characters without control characters")
		}
	}
	s, err := b.get(id)
	if err != nil {
		return Session{}, err
	}
	s.mu.Lock()
	if s.stopping || s.runtimeClosing {
		s.mu.Unlock()
		return Session{}, ErrUnavailable
	}
	if s.info.MetadataRevision != update.ExpectedRevision {
		s.mu.Unlock()
		return Session{}, ErrConflict
	}
	metadata := SessionMetadata{s.info.Title, s.info.Archived, s.info.MetadataRevision}
	if update.Title != nil {
		metadata.Title = *update.Title
	}
	if update.Archived != nil {
		metadata.Archived = *update.Archived
	}
	if metadata.Title == s.info.Title && metadata.Archived == s.info.Archived {
		info := s.info
		s.mu.Unlock()
		return info, nil
	}
	metadata.MetadataRevision++
	err = s.appendLocked(Event{Kind: "session.metadata", Metadata: &metadata})
	finish := func() {}
	if err == nil && metadata.Archived {
		finish, err = s.retireRuntimeLocked()
	}
	info, seq := s.info, s.info.LastSeq
	s.releaseHistoryLocked()
	s.mu.Unlock()
	finish()
	if err != nil {
		return info, err
	}
	// The change is acknowledged once it is on disk.
	return info, s.waitDurable(context.Background(), seq)
}

// Caller holds s.mu. Runtime teardown is fenced until Close has completed, so
// an old provider cannot publish into a newly resumed conversation.
func (s *liveSession) retireRuntimeLocked() (func(), error) {
	if s.runtimeClosing || s.stopping {
		return func() {}, ErrConflict
	}
	if s.info.State == "closed" {
		s.releaseHistoryLocked()
		return func() {}, nil
	}
	s.runtimeClosing = true
	driver, cancel, opening := s.driver, s.cancel, s.opening
	s.driver = nil
	for _, p := range s.activeInputs {
		if p.active {
			p.cancel()
		}
	}
	err := s.appendLocked(Event{Kind: "session.state", Status: "closed"})
	return func() {
		if cancel != nil {
			cancel()
		}
		if opening != nil {
			opening()
		}
		if driver != nil {
			_ = driver.Close()
		}
		s.mu.Lock()
		s.runtimeClosing = false
		s.releaseHistoryLocked()
		s.mu.Unlock()
	}, err
}

// Closed and disconnected history keeps only metadata in memory. Replay and
// idempotency checks read bounded records on demand, without retaining transcript content.
func (s *liveSession) releaseHistoryLocked() {
	if s.current != "" || s.opening != nil || (s.info.State != "closed" && s.info.State != "disconnected") {
		return
	}
	if len(s.activeInputs) > 0 {
		return
	}
	s.turns = map[string]turnRecord{}
	s.inputs = map[string]*pendingInput{}
	s.activeInputs = map[string]*pendingInput{}
	s.currentView = nil
}

// Must be called while b.mu is held, before locking the candidate session.
func (b *Broker) activeSessionsLocked() int {
	count := 0
	for _, s := range b.sessions {
		s.mu.Lock()
		if !s.info.Archived && s.info.State != "closed" && s.info.State != "disconnected" {
			count++
		}
		s.mu.Unlock()
	}
	return count
}

type SessionPage struct {
	Sessions   []Session `json:"sessions"`
	NextCursor string    `json:"nextCursor,omitempty"`
}

type SessionListOptions struct {
	Limit   int
	After   string
	Context ContextRef
}

// The cursor names an immutable creation boundary and the query scope. It
// remains valid across renames, archival and concurrent new conversations.
type sessionListCursor struct {
	CreatedAt string     `json:"createdAt"`
	ID        string     `json:"id"`
	Context   ContextRef `json:"context"`
}

// sessionBefore orders the immutable creation identities newest first.
func sessionBefore(a *liveSession, created, id string) bool {
	return a.info.CreatedAt > created || (a.info.CreatedAt == created && a.id > id)
}

// Caller holds b.mu (or is constructing the broker before publication).
func (b *Broker) indexSessionLocked(s *liveSession) {
	insert := func(index []*liveSession) []*liveSession {
		i := sort.Search(len(index), func(i int) bool { return !sessionBefore(index[i], s.info.CreatedAt, s.id) })
		index = append(index, nil)
		copy(index[i+1:], index[i:])
		index[i] = s
		return index
	}
	b.allSessions = insert(b.allSessions)
	ref := s.info.Scope.Context
	b.catalog[ref] = insert(b.catalog[ref])
}

func (b *Broker) List(ctx context.Context, options SessionListOptions) (SessionPage, error) {
	if options.Limit == 0 {
		options.Limit = 50
	}
	if options.Limit < 1 || options.Limit > 100 || ((options.Context.Kind == "") != (options.Context.ID == "")) ||
		(options.Context.Kind != "" && (!safeID.MatchString(options.Context.Kind) || !safeID.MatchString(options.Context.ID))) {
		return SessionPage{}, invalid("invalid conversation list scope or limit")
	}
	var cursor sessionListCursor
	if options.After != "" {
		if len(options.After) > 1400 {
			return SessionPage{}, invalid("invalid conversation list cursor")
		}
		data, err := base64.RawURLEncoding.DecodeString(options.After)
		if err != nil || len(data) > 1024 || json.Unmarshal(data, &cursor) != nil || !safeID.MatchString(cursor.ID) || cursor.CreatedAt == "" || cursor.Context != options.Context {
			return SessionPage{}, invalid("invalid conversation list cursor")
		}
	}
	if err := ctx.Err(); err != nil {
		return SessionPage{}, err
	}
	b.mu.Lock()
	index := b.allSessions
	if options.Context.Kind != "" {
		index = b.catalog[options.Context]
	}
	start := 0
	if cursor.ID != "" {
		start = sort.Search(len(index), func(i int) bool {
			return !sessionBefore(index[i], cursor.CreatedAt, cursor.ID) && (index[i].info.CreatedAt != cursor.CreatedAt || index[i].id != cursor.ID)
		})
	}
	end := min(len(index), start+options.Limit)
	more := end < len(index)
	page := SessionPage{Sessions: make([]Session, 0, end-start)}
	for _, s := range index[start:end] {
		s.mu.Lock()
		page.Sessions = append(page.Sessions, s.info)
		s.mu.Unlock()
	}
	b.mu.Unlock()
	if more {
		last := page.Sessions[len(page.Sessions)-1]
		data, _ := json.Marshal(sessionListCursor{last.CreatedAt, last.ID, options.Context})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	return page, nil
}

type PendingRequest struct {
	Request   Request `json:"request"`
	Submitted bool    `json:"submitted"`
}

func (b *Broker) Inputs(ctx context.Context, id string) ([]PendingRequest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b.mu.Lock()
	s := b.sessions[id]
	if s == nil || s.retiring {
		b.mu.Unlock()
		return nil, ErrNotFound
	}
	s.mu.Lock()
	b.mu.Unlock()
	defer s.mu.Unlock()
	items := []PendingRequest{}
	for _, p := range s.activeInputs {
		if p.active {
			items = append(items, PendingRequest{Request: p.request, Submitted: p.answer != nil})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Request.ID < items[j].Request.ID })
	return items, nil
}

type AttentionRequest struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt,omitempty"`
	Submitted bool   `json:"submitted"`
}

type AttentionSession struct {
	SessionID string             `json:"sessionId"`
	Context   ContextRef         `json:"context"`
	Workspace WorkspaceRef       `json:"workspace"`
	Worker    string             `json:"worker"`
	RuntimeID string             `json:"runtimeId,omitempty"`
	LastSeq   uint64             `json:"lastSeq"`
	Pending   []AttentionRequest `json:"pending"`
}

type AttentionSnapshot struct {
	Sessions []AttentionSession `json:"sessions"`
	Revision string             `json:"revision"`
}

type HistoryOptions struct {
	Before uint64
	Limit  int
}

type HistoryPage struct {
	Events     []Event `json:"events"`
	LastSeq    uint64  `json:"lastSeq"`
	NextBefore uint64  `json:"nextBefore,omitempty"`
}

func (b *Broker) History(ctx context.Context, id string, options HistoryOptions) (HistoryPage, error) {
	if options.Limit == 0 {
		options.Limit = replayPage
	}
	if options.Limit < 1 || options.Limit > replayPage {
		return HistoryPage{}, invalid("invalid history page limit")
	}
	if err := ctx.Err(); err != nil {
		return HistoryPage{}, err
	}
	s, release, err := b.retainReader(id, false)
	if err != nil {
		return HistoryPage{}, err
	}
	defer release()
	s.mu.Lock()
	page := HistoryPage{Events: []Event{}, LastSeq: s.info.LastSeq}
	if options.Before > page.LastSeq+1 {
		s.mu.Unlock()
		return HistoryPage{}, ErrCursor
	}
	end := page.LastSeq
	if options.Before > 0 {
		end = options.Before - 1
	}
	durable := s.durable
	if end > durable {
		bytes := 0
		for i := len(s.pending) - 1; i >= 0 && len(page.Events) < options.Limit; i-- {
			p := s.pending[i]
			if p.event.Seq > end {
				continue
			}
			if p.event.Seq <= durable || (len(page.Events) > 0 && bytes+len(p.data) > replayBytes) {
				break
			}
			page.Events = append(page.Events, p.event)
			bytes += len(p.data)
		}
		for i, j := 0, len(page.Events)-1; i < j; i, j = i+1, j-1 {
			page.Events[i], page.Events[j] = page.Events[j], page.Events[i]
		}
	}
	s.mu.Unlock()
	if end > 0 && end <= durable {
		n := min(options.Limit, int(end))
		page.Events, err = s.store.readHistory(ctx, id, end, n)
		if err != nil {
			return HistoryPage{}, err
		}
	}
	if len(page.Events) > 0 && page.Events[0].Seq > 1 {
		page.NextBefore = page.Events[0].Seq
	}
	return page, nil
}

// indexAttentionLocked is called on state/input transitions, not for ordinary
// output deltas. The index only contains sessions capable of needing attention.
func (s *liveSession) indexAttentionLocked() {
	if s.broker == nil {
		return
	}
	live := s.info.State != "closed" && s.info.State != "disconnected" && !s.info.Archived
	live = live || len(s.activeInputs) > 0
	if live {
		s.broker.attention.Store(s.id, s)
	} else {
		s.broker.attention.Delete(s.id)
	}
}

func (b *Broker) Attention(ctx context.Context) (AttentionSnapshot, error) {
	result := AttentionSnapshot{Sessions: []AttentionSession{}}
	b.attention.Range(func(_, value any) bool {
		if ctx.Err() != nil {
			return false
		}
		s := value.(*liveSession)
		s.mu.Lock()
		entry := AttentionSession{SessionID: s.info.ID, Context: s.info.Scope.Context, Workspace: s.info.Scope.Workspace, Worker: s.info.State, RuntimeID: s.info.RuntimeID, LastSeq: s.info.LastSeq, Pending: []AttentionRequest{}}
		for _, p := range s.activeInputs {
			if p.active {
				title := p.request.Title
				if len(title) > 512 {
					title = title[:512]
					for !utf8.ValidString(title) {
						title = title[:len(title)-1]
					}
					title += "…"
				}
				entry.Pending = append(entry.Pending, AttentionRequest{p.request.ID, p.request.Kind, title, p.request.CreatedAt, p.answer != nil})
			}
		}
		live := s.info.State != "closed" && s.info.State != "disconnected" && !s.info.Archived
		s.mu.Unlock()
		if live || len(entry.Pending) > 0 {
			sort.Slice(entry.Pending, func(i, j int) bool { return entry.Pending[i].ID < entry.Pending[j].ID })
			result.Sessions = append(result.Sessions, entry)
		}
		return true
	})
	if err := ctx.Err(); err != nil {
		return AttentionSnapshot{}, err
	}
	sort.Slice(result.Sessions, func(i, j int) bool { return result.Sessions[i].SessionID < result.Sessions[j].SessionID })
	data, _ := json.Marshal(result.Sessions)
	result.Revision = hash(string(data))
	return result, nil
}

type DecisionActor struct {
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

type DecisionPolicy struct {
	ID       string
	Revision string
}

type DecisionReceipt struct {
	OptionID       string         `json:"optionId,omitempty"`
	OptionKind     string         `json:"optionKind,omitempty"`
	Outcome        string         `json:"outcome"`
	Actor          *DecisionActor `json:"actor,omitempty"`
	PolicyID       string         `json:"policyId,omitempty"`
	PolicyRevision string         `json:"policyRevision,omitempty"`
}

type decisionActorKey struct{}
type decisionPolicyKey struct{}

// These hooks are for authenticated host composition. Request bodies and
// provider tool arguments cannot populate the actor or policy in an audit receipt.
func WithDecisionActor(ctx context.Context, actor DecisionActor) context.Context {
	return context.WithValue(ctx, decisionActorKey{}, actor)
}

func WithDecisionPolicy(ctx context.Context, policy DecisionPolicy) context.Context {
	return context.WithValue(ctx, decisionPolicyKey{}, policy)
}

func decisionReceipt(ctx context.Context, request Request, answer Answer) *DecisionReceipt {
	receipt := &DecisionReceipt{Outcome: "answer"}
	if answer.Cancel {
		receipt.Outcome = "cancel"
	} else if len(request.Questions) == 0 {
		for _, option := range request.Options {
			if option.ID == answer.OptionID {
				receipt.OptionID, receipt.OptionKind = option.ID, option.Kind
				if strings.HasPrefix(option.Kind, "allow") {
					receipt.Outcome = "allow"
				} else if strings.HasPrefix(option.Kind, "reject") {
					receipt.Outcome = "deny"
				}
				break
			}
		}
	}
	if actor, ok := ctx.Value(decisionActorKey{}).(DecisionActor); ok && actor.ID != "" {
		receipt.Actor = &actor
	}
	if policy, ok := ctx.Value(decisionPolicyKey{}).(DecisionPolicy); ok {
		receipt.PolicyID, receipt.PolicyRevision = policy.ID, policy.Revision
	}
	return receipt
}
