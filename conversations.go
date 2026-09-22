package nativeagent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Session.ID is the durable conversation identity. Metadata changes and worker
// restarts append to its existing journal; they never create another transcript.
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

func (b *Broker) UpdateMetadata(id string, update MetadataUpdate) (Session, error) {
	if update.ExpectedRevision == 0 || (update.Title == nil && update.Archived == nil) {
		return Session{}, errors.New("metadata revision and an explicit change are required")
	}
	if update.Title != nil {
		title := *update.Title
		if !utf8.ValidString(title) || strings.TrimSpace(title) != title || utf8.RuneCountInString(title) > 160 || strings.ContainsFunc(title, unicode.IsControl) {
			return Session{}, errors.New("conversation title must be at most 160 characters without control characters")
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
	info := s.info
	s.releaseJournalLocked()
	s.mu.Unlock()
	finish()
	return info, err
}

// Caller holds s.mu. Runtime teardown is fenced until Close has completed, so
// an old provider cannot publish into a newly resumed conversation.
func (s *liveSession) retireRuntimeLocked() (func(), error) {
	if s.runtimeClosing {
		return func() {}, ErrConflict
	}
	if s.info.State == "closed" {
		s.releaseJournalLocked()
		return func() {}, nil
	}
	s.runtimeClosing = true
	driver, cancel, opening := s.driver, s.cancel, s.opening
	s.driver = nil
	for _, p := range s.inputs {
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
		s.releaseJournalLocked()
		s.mu.Unlock()
	}, err
}

// Closed and disconnected history keeps only metadata in memory. Replay and
// idempotency checks read bounded records on demand, without a writer per log.
func (s *liveSession) releaseJournalLocked() {
	if s.current != "" || s.opening != nil || (s.info.State != "closed" && s.info.State != "disconnected") {
		return
	}
	for _, input := range s.inputs {
		if input.active {
			return
		}
	}
	if s.file != nil {
		_ = s.file.Close()
		s.file = nil
	}
	s.events = nil
	s.turns = map[string]string{}
	s.inputs = map[string]*pendingInput{}
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

func (b *Broker) ListPage(options SessionListOptions) (SessionPage, error) {
	if options.Limit == 0 {
		options.Limit = 50
	}
	if options.Limit < 1 || options.Limit > 100 || ((options.Context.Kind == "") != (options.Context.ID == "")) ||
		(options.Context.Kind != "" && (!safeID.MatchString(options.Context.Kind) || !safeID.MatchString(options.Context.ID))) {
		return SessionPage{}, errors.New("invalid conversation list scope or limit")
	}
	var cursor sessionListCursor
	if options.After != "" {
		data, err := base64.RawURLEncoding.DecodeString(options.After)
		if err != nil || len(data) > 1024 || json.Unmarshal(data, &cursor) != nil || !safeID.MatchString(cursor.ID) || cursor.CreatedAt == "" || cursor.Context != options.Context {
			return SessionPage{}, errors.New("invalid conversation list cursor")
		}
	}
	page := SessionPage{Sessions: []Session{}}
	for _, session := range b.List() {
		if options.Context.Kind != "" && session.Scope.Context != options.Context {
			continue
		}
		if cursor.ID != "" && (session.CreatedAt > cursor.CreatedAt || (session.CreatedAt == cursor.CreatedAt && session.ID >= cursor.ID)) {
			continue
		}
		if len(page.Sessions) == options.Limit {
			last := page.Sessions[len(page.Sessions)-1]
			data, _ := json.Marshal(sessionListCursor{last.CreatedAt, last.ID, options.Context})
			page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
			break
		}
		page.Sessions = append(page.Sessions, session)
	}
	return page, nil
}

type PendingRequest struct {
	Request   Request        `json:"request"`
	Submitted bool           `json:"submitted"`
	Facts     *DecisionFacts `json:"facts,omitempty"`
}

func (b *Broker) Inputs(id string) ([]PendingRequest, error) {
	s, err := b.get(id)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []PendingRequest{}
	for _, p := range s.inputs {
		if p.active {
			item := PendingRequest{Request: p.request, Submitted: p.answer != nil}
			if facts, known := nativeDecisionFacts(s.info, p.request, s.cwd); known {
				item.Facts = &facts
			}
			items = append(items, item)
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

func (b *Broker) Attention() AttentionSnapshot {
	b.mu.Lock()
	sessions := make([]*liveSession, 0, len(b.sessions))
	for _, s := range b.sessions {
		sessions = append(sessions, s)
	}
	b.mu.Unlock()
	result := AttentionSnapshot{Sessions: []AttentionSession{}}
	for _, s := range sessions {
		s.mu.Lock()
		entry := AttentionSession{SessionID: s.info.ID, Context: s.info.Scope.Context, Workspace: s.info.Scope.Workspace, Worker: s.info.State, RuntimeID: s.info.RuntimeID, LastSeq: s.info.LastSeq, Pending: []AttentionRequest{}}
		for _, p := range s.inputs {
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
	}
	sort.Slice(result.Sessions, func(i, j int) bool { return result.Sessions[i].SessionID < result.Sessions[j].SessionID })
	data, _ := json.Marshal(result.Sessions)
	result.Revision = hash(string(data))
	return result
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

// These hooks are for authenticated host composition. HTTP bodies and provider
// tool arguments cannot populate the actor or policy in an audit receipt.
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
