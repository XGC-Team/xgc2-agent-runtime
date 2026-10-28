package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/XGC-Team/xgc2-storage/api"
	xrpc "github.com/XGC-Team/xgc2-xrpc/go"
)

// Storage also enforces its own collection and receipt quotas. These are
// logical limits on the runtime-owned event JSON, including uncommitted events.
const (
	MaxRetainedHistoryBytes = int64(384 << 20)
	retentionSoftBytes      = int64(320 << 20)
	retentionSoftSessions   = 1800
	HistoryMaxAge           = 30 * 24 * time.Hour
	retentionStepEvents     = 16
	retentionStepsPerCall   = 16
	maxHistoryReaders       = 32
	maxReadersPerSession    = 8
)

type HistoryCleanup struct {
	DeletedEvents int  `json:"deletedEvents"`
	Complete      bool `json:"complete"`
}

func (b *Broker) reserveHistory(size int64) bool {
	for {
		before := b.historyBytes.Load()
		if size > MaxRetainedHistoryBytes-before {
			return false
		}
		if b.historyBytes.CompareAndSwap(before, before+size) {
			if before < retentionSoftBytes && before+size >= retentionSoftBytes {
				// The session lock is held here. The scheduling goroutine takes the
				// broker lock only after this append can release the session lock.
				go b.scheduleRetention()
			}
			return true
		}
	}
}

// retainReader fences retirement through the complete read or subscription.
func (b *Broker) retainReader(id string, subscriber bool) (*liveSession, func(), error) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil, nil, ErrUnavailable
	}
	s := b.sessions[id]
	if s == nil || s.retiring {
		b.mu.Unlock()
		return nil, nil, ErrNotFound
	}
	s.mu.Lock()
	if b.readers >= maxHistoryReaders || s.readers >= maxReadersPerSession {
		s.mu.Unlock()
		b.mu.Unlock()
		return nil, nil, exhausted("history reader capacity reached")
	}
	b.readers++
	s.readers++
	if subscriber {
		s.subscribers++
		s.seedLeaseUntil = time.Time{}
	}
	s.mu.Unlock()
	b.mu.Unlock()
	return s, func() {
		b.mu.Lock()
		s.mu.Lock()
		s.readers--
		b.readers--
		if subscriber {
			s.subscribers--
		}
		updated, err := time.Parse(time.RFC3339Nano, s.updatedAt)
		needed := b.retiringCount > 0 || len(b.sessions) >= retentionSoftSessions || b.historyBytes.Load() >= retentionSoftBytes || (err == nil && time.Since(updated) >= HistoryMaxAge)
		s.mu.Unlock()
		b.mu.Unlock()
		if needed {
			b.scheduleRetention()
		}
	}, nil
}

// removableLocked checks actual unsettled relationships, not whether an idle
// conversation happens to have an open provider process.
func (s *liveSession) removableLocked() bool {
	if s.current != "" || s.opening != nil || s.runtimeClosing || s.readers != 0 || time.Now().Before(s.seedLeaseUntil) || len(s.queue.Items) != 0 || s.persistErr != nil {
		return false
	}
	return len(s.activeInputs) == 0
}

func (b *Broker) unindexSessionLocked(s *liveSession) {
	remove := func(index []*liveSession) []*liveSession {
		for i, item := range index {
			if item == s {
				copy(index[i:], index[i+1:])
				index[len(index)-1] = nil
				return index[:len(index)-1]
			}
		}
		return index
	}
	b.allSessions = remove(b.allSessions)
	ref := s.info.Scope.Context
	b.catalog[ref] = remove(b.catalog[ref])
	if len(b.catalog[ref]) == 0 {
		delete(b.catalog, ref)
	}
	b.attention.Delete(s.id)
}

// retirementCandidateLocked is called only after a lifecycle or quota event.
// No timer, empty-history scanner or observer drives retention.
func (b *Broker) retirementCandidateLocked(now time.Time) *liveSession {
	if b.retiringCount > 0 {
		for _, s := range b.sessions {
			if s.retiring {
				return s
			}
		}
	}
	pressure := len(b.sessions) >= retentionSoftSessions || b.historyBytes.Load() >= retentionSoftBytes
	var candidate *liveSession
	var oldest time.Time
	for i := len(b.allSessions) - 1; i >= 0; i-- {
		s := b.allSessions[i]
		s.mu.Lock()
		updated, err := time.Parse(time.RFC3339Nano, s.updatedAt)
		expired := err == nil && now.Sub(updated) >= HistoryMaxAge
		ok := (pressure || expired) && s.removableLocked() && !s.stopping
		s.mu.Unlock()
		if ok && (candidate == nil || updated.Before(oldest)) {
			candidate, oldest = s, updated
		}
	}
	return candidate
}

func (b *Broker) scheduleRetention() {
	b.mu.Lock()
	now := time.Now()
	potential := b.retiringCount > 0 || len(b.sessions) >= retentionSoftSessions || b.historyBytes.Load() >= retentionSoftBytes
	if !potential && len(b.allSessions) > 0 {
		oldest, err := time.Parse(time.RFC3339Nano, b.allSessions[len(b.allSessions)-1].info.CreatedAt)
		potential = err == nil && now.Sub(oldest) >= HistoryMaxAge
	}
	if b.closed || b.retentionRunning || !potential || b.retirementCandidateLocked(now) == nil {
		b.mu.Unlock()
		return
	}
	b.retentionRunning = true
	b.wg.Add(1)
	b.mu.Unlock()
	go func() {
		defer b.wg.Done()
		defer func() {
			b.mu.Lock()
			b.retentionRunning = false
			b.mu.Unlock()
		}()
		select {
		case b.retentionSlot <- struct{}{}:
		case <-b.ctx.Done():
			return
		}
		defer func() { <-b.retentionSlot }()
		for step := 0; step < retentionStepsPerCall; step++ {
			if b.ctx.Err() != nil {
				return
			}
			b.mu.Lock()
			s := b.retirementCandidateLocked(time.Now())
			b.mu.Unlock()
			if s == nil {
				return
			}
			if _, _, err := b.retireStep(b.ctx, s.id); err != nil {
				return
			}
		}
	}()
}

// CleanupHistory performs finite work and returns whether the named session was
// completely retired. Further calls resume a partially retired session. Active
// turns, pending decisions, queued prompts and current readers are protected.
func (b *Broker) CleanupHistory(ctx context.Context, id string) (HistoryCleanup, error) {
	if err := ctx.Err(); err != nil {
		return HistoryCleanup{}, err
	}
	if b.ctx.Err() != nil {
		return HistoryCleanup{}, ErrUnavailable
	}
	select {
	case b.retentionSlot <- struct{}{}:
	default:
		return HistoryCleanup{}, exhausted("history cleanup is already in progress")
	}
	defer func() { <-b.retentionSlot }()
	if err := ctx.Err(); err != nil {
		return HistoryCleanup{}, err
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return HistoryCleanup{}, ErrUnavailable
	}
	b.wg.Add(1)
	b.mu.Unlock()
	defer b.wg.Done()
	caller := ctx
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(b.ctx, cancel)
	defer func() { stop(); cancel() }()
	result := HistoryCleanup{}
	for i := 0; i < retentionStepsPerCall; i++ {
		deleted, complete, err := b.retireStep(ctx, id)
		result.DeletedEvents += deleted
		result.Complete = complete
		if err != nil || complete {
			if errors.Is(err, context.Canceled) && caller.Err() == nil && b.ctx.Err() != nil {
				err = ErrUnavailable
			}
			return result, err
		}
	}
	return result, nil
}

// retireStep owns one Storage batch. The session record itself records progress,
// so a crash cannot make a partly deleted transcript look complete on reopen.
func (b *Broker) retireStep(ctx context.Context, id string) (int, bool, error) {
	if err := ctx.Err(); err != nil {
		return 0, false, err
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return 0, false, ErrUnavailable
	}
	s := b.sessions[id]
	if s == nil {
		b.mu.Unlock()
		return 0, false, ErrNotFound
	}
	s.mu.Lock()
	var cancel context.CancelFunc
	if !s.retiring {
		if s.stopping || !s.removableLocked() {
			s.mu.Unlock()
			b.mu.Unlock()
			return 0, false, ErrConflict
		}
		s.retiring, s.stopping = true, true
		b.retiringCount++
		s.info.State = "disconnected"
		s.retireDriver, cancel = s.driver, s.cancel
		s.driver, s.cancel = nil, nil
		b.unindexSessionLocked(s)
	}
	s.startDriverCloseLocked(s.retireDriver)
	stopped := s.stopClose
	s.mu.Unlock()
	b.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if stopped != nil {
		select {
		case <-stopped:
		case <-ctx.Done():
			return 0, false, ctx.Err()
		}
		s.mu.Lock()
		closeErr := s.stopCloseErr
		if closeErr == nil {
			s.retireDriver = nil
		}
		s.mu.Unlock()
		if closeErr != nil {
			return 0, false, closeErr
		}
	}
	if err := s.flushAll(ctx); err != nil {
		return 0, false, err
	}
	s.mu.Lock()
	version, through, persisted, last := s.version, s.retiredThrough, s.retirementPersisted, s.info.LastSeq
	s.mu.Unlock()
	newVersion, removed, bytes, complete, err := s.store.retireEvents(ctx, id, version, through, persisted, last)
	if err != nil {
		return 0, false, err
	}
	s.mu.Lock()
	s.version = newVersion
	s.retiredThrough += uint64(removed)
	s.retirementPersisted = !complete
	s.bytes -= bytes
	s.mu.Unlock()
	b.historyBytes.Add(-bytes)
	if complete {
		b.mu.Lock()
		if b.sessions[id] == s {
			delete(b.sessions, id)
			b.retiringCount--
		}
		b.mu.Unlock()
	}
	return removed, complete, nil
}

func resourceExhausted(err error) bool {
	var domain *api.Error
	var call *xrpc.CallError
	return (errors.As(err, &domain) && domain.Code == "resource_exhausted") || (errors.As(err, &call) && call.Code == "resource_exhausted")
}

// retireEvents atomically removes a bounded consecutive group, its prompt
// identities, and updates (or removes) the same session CAS anchor.
func (s *Store) retireEvents(ctx context.Context, id, version string, through uint64, marked bool, last uint64) (string, int, int64, bool, error) {
	ctx, release, err := s.begin(ctx)
	if err != nil {
		return "", 0, 0, false, err
	}
	defer release()
	n := int(min(uint64(retentionStepEvents), last-through))
	var read api.SnapshotResponse
	for {
		queries := []api.Query{{Collection: "sessions", Keys: []string{id}}}
		if n > 0 {
			keys := make([]string, n)
			for i := range keys {
				keys[i] = eventKey(id, through+1+uint64(i))
			}
			queries = append(queries, api.Query{Collection: "events", Keys: keys})
		}
		read, err = s.snapshot(ctx, queries, nil)
		if err != nil && n > 1 && resourceExhausted(err) {
			n = (n + 1) / 2
			continue
		}
		if err != nil {
			return "", 0, 0, false, err
		}
		break
	}
	row := read.Results[0].Records[0]
	var record sessionRecord
	if row.Missing || row.Deleted || row.Version != version || json.Unmarshal(row.Data, &record) != nil || record.ID != id || record.LastSeq != last || record.Retiring != marked || record.RetiredThrough != through {
		return "", 0, 0, false, ErrConflict
	}
	mutations := make([]api.Mutation, 0, 1+2*n)
	var removedBytes int64
	var promptIDs []string
	if n > 0 {
		rows := read.Results[1].Records
		events, decodeErr := decodeEventRows(id, through+1, rows)
		if decodeErr != nil {
			return "", 0, 0, false, decodeErr
		}
		for i, event := range events {
			removedBytes += int64(len(rows[i].Data))
			mutations = append(mutations, api.Mutation{Collection: "events", Key: rows[i].Key, ExpectedVersion: rows[i].Version, Delete: true})
			if event.Kind == "item.snapshot" && event.Role == "user" && event.Status == "submitted" {
				promptIDs = append(promptIDs, event.TurnID)
			}
		}
	}
	if removedBytes > record.Bytes {
		return "", 0, 0, false, errors.New("agent-runtime: invalid retained event bytes")
	}
	if len(promptIDs) > 0 {
		prompts, err := s.snapshot(ctx, []api.Query{{Collection: "prompts", Keys: promptIDs}}, &read.Token)
		if err != nil {
			return "", 0, 0, false, err
		}
		seen := map[string]bool{}
		for _, prompt := range prompts.Results[0].Records {
			var value struct {
				SessionID   string `json:"sessionId"`
				Fingerprint string `json:"fingerprint"`
			}
			if seen[prompt.Key] || prompt.Missing || prompt.Deleted || json.Unmarshal(prompt.Data, &value) != nil || value.SessionID != id || !digest.MatchString(value.Fingerprint) {
				return "", 0, 0, false, errors.New("agent-runtime: retained prompt identity unavailable")
			}
			seen[prompt.Key] = true
			mutations = append(mutations, api.Mutation{Collection: "prompts", Key: prompt.Key, ExpectedVersion: prompt.Version, Delete: true})
		}
	}
	complete := through+uint64(n) == last
	if complete && record.Bytes != removedBytes {
		return "", 0, 0, false, errors.New("agent-runtime: retained event byte total does not match")
	}
	anchor := api.Mutation{Collection: "sessions", Key: id, ExpectedVersion: version, Delete: complete}
	if !complete {
		record.Retiring = true
		record.RetiredThrough = through + uint64(n)
		record.Bytes -= removedBytes
		record.State = "disconnected"
		anchor.Data, err = json.Marshal(record)
		if err != nil {
			return "", 0, 0, false, err
		}
	}
	mutations = append([]api.Mutation{anchor}, mutations...)
	s.token = nil
	receipt, err := s.commit(ctx, read.Token, mutations)
	if err != nil {
		return "", 0, 0, false, err
	}
	s.token = &receipt.Token
	return receipt.Token.Revision, n, removedBytes, complete, nil
}
