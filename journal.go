package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/XGC-Team/xgc2-storage/api"
)

// The journal of a conversation is its events in sequence order. A new event is
// numbered, applied to the live state and published to subscribers at once. Its
// commit to storage follows within the journal delay, in one batch with the
// events that arrived meanwhile and with the conversation record they leave
// behind. Events that must not be acted on before they are durable are
// committed without delay, and the action waits for them. No lock is held
// across a storage call.
const (
	// maxBatchEvents keeps a batch (record, events and prompt identities) within
	// api.MaxOperations; maxBatchBytes keeps it within api.MaxRequestBytes next
	// to the largest single event.
	maxBatchEvents = 200
	maxBatchBytes  = 3 << 20
	// A native client that produces events faster than they can be committed is
	// held back at these bounds instead of growing the pending list.
	maxPendingEvents = 512
	maxPendingBytes  = 2 << 20
	maxEventBytes    = 3 << 20
	// All appenders share these admission limits, including the batch currently
	// being committed (it remains in pending until the receipt is confirmed).
	maxBufferedEvents = 4096
	maxBufferedBytes  = maxPendingBytes + maxEventBytes
)

// pendingEvent is an event numbered but not yet committed.
type pendingEvent struct {
	event Event
	data  []byte
	// record is the conversation as it is after this event, so a batch can end
	// at any event.
	record sessionRecord
	at     time.Time
	urgent bool
}

// durableKind reports whether e is committed without waiting for company: the
// events a caller or a native client must not act before. Streamed output,
// notices and worker state changes are batched.
func durableKind(e Event) bool {
	switch e.Kind {
	case "item.delta", "item.patch", "notice", "session.state", "input.request":
		return false
	case "item.snapshot":
		return e.Role == "user" && e.Status == "submitted"
	}
	return true
}

// appendLocked numbers e, applies it to the live conversation and queues it for
// commit. It fails when the conversation has no capacity left or has already
// failed; a storage failure surfaces later, through waitDurable.
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
	useReservation := s.reservedControl > 0 && e.Kind == "session.state" && e.Status == "running" && e.TurnID == s.current
	reservedBytes := s.reservedControl
	reservedEvents := 0
	if useReservation {
		reservedBytes = 0
		if int64(len(data)) > s.reservedControl {
			return s.stopLocked(exhausted("running-state event exceeds its admission reservation"))
		}
	} else if reservedBytes > 0 {
		reservedEvents = 1
	}
	if len(data) > maxEventBytes || s.bytes+int64(len(data))+reservedBytes > MaxSessionBytes || s.info.LastSeq+uint64(1+reservedEvents) > MaxEvents {
		return s.stopLocked(exhausted("event storage capacity reached"))
	}
	if len(s.pending)+1+reservedEvents > maxBufferedEvents || s.pendingBytes+int64(len(data))+reservedBytes > maxBufferedBytes {
		return s.stopLocked(exhausted("event journal admission capacity reached"))
	}
	if useReservation {
		if s.broker != nil {
			s.broker.historyBytes.Add(-(s.reservedControl - int64(len(data))))
		}
		s.reservedControl = 0
	} else if s.broker != nil && !s.broker.reserveHistory(int64(len(data))) {
		return s.stopLocked(exhausted("retained history byte capacity reached"))
	}
	queued := false
	if e.Kind == "item.snapshot" && e.Role == "user" && e.Status == "submitted" && s.queueSeq > 0 {
		for _, item := range s.queue.Items {
			queued = queued || item.ID == e.TurnID
		}
	}
	s.bytes += int64(len(data))
	s.updatedAt = e.CreatedAt
	s.info.LastSeq = e.Seq
	if e.Kind == "session.state" {
		s.info.State = e.Status
	}
	if e.Kind == "session.identity" {
		s.info.AgentSessionID = e.AgentSessionID
	}
	s.applyMetadataLocked(e)
	s.applyCurrentLocked(e, len(data))
	if e.Kind == "session.state" || e.Kind == "session.metadata" || e.Kind == "input.request" || e.Kind == "input.resolved" {
		s.indexAttentionLocked()
	}
	if e.Kind == "prompt.queue" {
		s.queueSeq = e.Seq
		s.queueBytes = len(data)
		s.submittedQueueTurns = nil
	}
	if queued {
		s.submittedQueueTurns = append(s.submittedQueueTurns, e.TurnID)
	}
	urgent := durableKind(e)
	s.pending = append(s.pending, pendingEvent{event: e, data: data, record: s.record(), at: time.Now(), urgent: urgent})
	s.pendingBytes += int64(len(data))
	s.scheduleLocked(urgent)
	s.broadcastLocked()
	return nil
}

// broadcastLocked wakes the subscribers: there is a new event or a failure.
func (s *liveSession) broadcastLocked() {
	if s.subscribers == 0 {
		return
	}
	close(s.changed)
	s.changed = make(chan struct{})
}

// stopLocked ends the conversation's ability to take events and stops its
// worker, which could no longer be recorded. Subscribers read what is pending
// and then see the failure.
func (s *liveSession) stopLocked(cause error) error {
	s.storageErr = errors.Join(errors.New("event journal stopped"), cause)
	s.info.State = "disconnected"
	s.indexAttentionLocked()
	for _, p := range s.activeInputs {
		if p.active {
			p.cancel()
		}
	}
	if s.cancel != nil {
		s.cancel()
	}
	s.startDriverCloseLocked(s.driver)
	s.broadcastLocked()
	return s.storageErr
}

// Caller holds the session lock. Exactly one close may be detached while its
// process joins; retirement and Broker.Close wait for this same completion.
func (s *liveSession) startDriverCloseLocked(driver Driver) {
	if driver != nil && s.stopClose == nil {
		done := make(chan struct{})
		s.stopClose = done
		go func() {
			err := driver.Close()
			s.mu.Lock()
			s.stopCloseErr = err
			s.mu.Unlock()
			close(done)
		}()
	}
}

// scheduleLocked makes sure a flush is under way; urgent skips the delay.
func (s *liveSession) scheduleLocked(urgent bool) {
	if s.persistErr != nil || len(s.pending) == 0 {
		return
	}
	if urgent {
		s.flushNow = true
	}
	if s.flushing {
		if urgent {
			select {
			case s.wake <- struct{}{}:
			default:
			}
		}
		return
	}
	s.flushing = true
	go s.flush()
}

// dueLocked is how long the oldest pending event may still wait for company.
func (s *liveSession) dueLocked() time.Duration {
	if s.flushNow || len(s.pending) >= maxBatchEvents || s.pendingBytes >= maxBatchBytes {
		return 0
	}
	for _, p := range s.pending {
		if p.urgent {
			return 0
		}
	}
	return s.store.JournalDelay - time.Since(s.pending[0].at)
}

// takeLocked copies the next batch off the pending list.
func (s *liveSession) takeLocked() ([]pendingEvent, int64) {
	var size int64
	operations := 1 // the conversation record
	n := 0
	for _, p := range s.pending {
		cost := 1
		if p.event.Kind == "item.snapshot" && p.event.Role == "user" && p.event.Status == "submitted" {
			cost = 2 // and the identity of the prompt
		}
		if n > 0 && (n >= maxBatchEvents || size+int64(len(p.data)) > maxBatchBytes || operations+cost > api.MaxOperations) {
			break
		}
		n++
		size += int64(len(p.data))
		operations += cost
	}
	return append([]pendingEvent(nil), s.pending[:n]...), size
}

// flush commits the pending events, batch by batch, until none is left or a
// commit fails. At most one flush runs per conversation.
func (s *liveSession) flush() {
	for {
		s.mu.Lock()
		if len(s.pending) == 0 || s.persistErr != nil {
			s.flushing, s.flushNow = false, false
			s.mu.Unlock()
			return
		}
		if wait := s.dueLocked(); wait > 0 {
			s.mu.Unlock()
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-s.wake:
			}
			timer.Stop()
			continue
		}
		batch, size := s.takeLocked()
		version := s.version
		s.flushNow = false
		s.mu.Unlock()

		newVersion, err := s.store.commitEvents(context.Background(), s.id, version, batch)

		s.mu.Lock()
		if err != nil {
			s.flushing = false
			s.persistErr = err
			s.stopLocked(err)
			close(s.flushed)
			s.flushed = make(chan struct{})
			s.mu.Unlock()
			return
		}
		s.version = newVersion
		s.durable = batch[len(batch)-1].event.Seq
		s.pendingBytes -= size
		clear(s.pending[:len(batch)])
		if s.pending = s.pending[len(batch):]; len(s.pending) == 0 {
			s.pending = nil
		}
		batch = nil
		close(s.flushed)
		s.flushed = make(chan struct{})
		s.mu.Unlock()
	}
}

// waitDurable returns once every event up to seq is committed, or with the
// error that stopped the conversation. Call it without holding any lock.
func (s *liveSession) waitDurable(ctx context.Context, seq uint64) error {
	s.mu.Lock()
	for {
		if s.durable >= seq {
			s.mu.Unlock()
			return nil
		}
		if s.persistErr != nil {
			err := s.storageErr
			s.mu.Unlock()
			return err
		}
		flushed := s.flushed
		s.scheduleLocked(true)
		s.mu.Unlock()
		select {
		case <-flushed:
		case <-ctx.Done():
			return ctx.Err()
		}
		s.mu.Lock()
	}
}

// waitCapacity holds a native client back while too many of its events wait for
// their commit.
func (s *liveSession) waitCapacity() {
	s.mu.Lock()
	for s.persistErr == nil && s.storageErr == nil && (len(s.pending) >= maxPendingEvents || s.pendingBytes >= maxPendingBytes) {
		flushed := s.flushed
		s.scheduleLocked(true)
		s.mu.Unlock()
		<-flushed
		s.mu.Lock()
	}
	s.mu.Unlock()
}

// flushAll commits everything the conversation has appended so far.
func (s *liveSession) flushAll(ctx context.Context) error {
	s.mu.Lock()
	seq := s.info.LastSeq
	s.mu.Unlock()
	return s.waitDurable(ctx, seq)
}

// tailLocked returns the events after the cursor, up to end, that are not yet in
// storage, and the highest sequence number that is.
func (s *liveSession) tailLocked(after, end uint64) ([]Event, []int) {
	var events []Event
	var sizes []int
	var bytes int
	for _, p := range s.pending {
		if p.event.Seq > after && p.event.Seq <= end {
			if len(events) > 0 && bytes+len(p.data) > replayBytes {
				break
			}
			events = append(events, p.event)
			sizes = append(sizes, len(p.data))
			bytes += len(p.data)
		}
	}
	return events, sizes
}
