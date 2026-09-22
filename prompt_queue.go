package nativeagent

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const maxQueuedPrompts = 20

type QueuedPrompt struct {
	ID             string        `json:"id"`
	Text           string        `json:"text"`
	Options        NativeOptions `json:"options"`
	RequestOptions NativeOptions `json:"requestOptions"`
	CreatedAt      string        `json:"createdAt"`
}
type PromptQueue struct {
	Revision uint64         `json:"revision"`
	Paused   bool           `json:"paused"`
	Items    []QueuedPrompt `json:"items"`
}
type QueueCommand struct {
	Operation        string        `json:"operation"`
	ExpectedRevision uint64        `json:"expectedRevision"`
	ID               string        `json:"id,omitempty"`
	Text             string        `json:"text,omitempty"`
	Options          NativeOptions `json:"options,omitempty"`
	Order            []string      `json:"order,omitempty"`
}

func validQueuedText(text string) bool {
	return strings.TrimSpace(text) != "" && len(text) <= 128<<10 && utf8.ValidString(text)
}
func cloneQueue(q PromptQueue) PromptQueue { q.Items = append([]QueuedPrompt{}, q.Items...); return q }
func (s *liveSession) writeQueueLocked(q PromptQueue) error {
	q.Revision = s.queue.Revision + 1
	return s.appendLocked(Event{Kind: "prompt.queue", Queue: &q})
}
func (s *liveSession) pauseQueueLocked() {
	if len(s.queue.Items) > 0 && !s.queue.Paused {
		q := cloneQueue(s.queue)
		q.Paused = true
		_ = s.writeQueueLocked(q)
	}
}
func (s *liveSession) applyQueueEventLocked(e Event) {
	if e.Kind == "prompt.queue" && e.Queue != nil {
		s.queue = cloneQueue(*e.Queue)
	}
	if e.Kind == "turn.end" {
		s.lastTurnStatus = e.Status
	}
	if e.Kind == "item.snapshot" && e.Role == "user" && e.Status == "submitted" {
		// A durable submission wins over a pending queue snapshot even if the host
		// crashed before recording the following queue revision. Never resend it.
		for i, p := range s.queue.Items {
			if p.ID == e.TurnID {
				q := cloneQueue(s.queue)
				q.Items = append(q.Items[:i], q.Items[i+1:]...)
				s.queue = q
				break
			}
		}
	}
}
func (b *Broker) Queue(id, key string, c QueueCommand) (PromptQueue, error) {
	if err := validKey(key); err != nil {
		return PromptQueue{}, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return PromptQueue{}, ErrUnavailable
	}
	s := b.sessions[id]
	if s == nil {
		return PromptQueue{}, ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.info.Archived {
		return PromptQueue{}, ErrUnavailable
	}
	q := cloneQueue(s.queue)
	if c.Operation != "enqueue" && c.ExpectedRevision != q.Revision {
		return PromptQueue{}, ErrConflict
	}
	switch c.Operation {
	case "enqueue":
		if !validQueuedText(c.Text) {
			return PromptQueue{}, errors.New("prompt must be nonempty UTF-8 up to 128 KiB")
		}
		turn := "t_" + hash(id + "\x00" + key)[:32]
		for _, p := range q.Items {
			if p.ID == turn {
				if p.Text != c.Text || p.RequestOptions != c.Options {
					return PromptQueue{}, ErrConflict
				}
				return q, nil
			}
		}
		fingerprint, found, err := s.findTurnLocked(turn)
		if err != nil {
			return PromptQueue{}, err
		}
		if found {
			if fingerprint != promptFingerprint(c.Text, c.Options) {
				return PromptQueue{}, ErrConflict
			}
			return q, nil
		}
		if len(q.Items) >= maxQueuedPrompts {
			return PromptQueue{}, errors.New("message queue is full")
		}
		selected, err := b.selection(s.profile, c.Options)
		if err != nil {
			return PromptQueue{}, err
		}
		if s.driver == nil || s.info.State == "disconnected" || s.info.State == "closed" {
			q.Paused = true
		}
		q.Items = append(q.Items, QueuedPrompt{ID: turn, Text: c.Text, Options: selected, RequestOptions: c.Options, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	case "edit", "remove":
		index := -1
		for i, p := range q.Items {
			if p.ID == c.ID {
				index = i
				break
			}
		}
		if index < 0 {
			return PromptQueue{}, ErrNotFound
		}
		if c.Operation == "edit" {
			if !validQueuedText(c.Text) {
				return PromptQueue{}, errors.New("invalid queued message")
			}
			q.Items[index].Text = c.Text
		} else {
			q.Items = append(q.Items[:index], q.Items[index+1:]...)
		}
	case "reorder":
		if len(c.Order) != len(q.Items) {
			return PromptQueue{}, ErrConflict
		}
		byID := map[string]QueuedPrompt{}
		for _, p := range q.Items {
			byID[p.ID] = p
		}
		reordered := []QueuedPrompt{}
		for _, id := range c.Order {
			p, ok := byID[id]
			if !ok {
				return PromptQueue{}, ErrConflict
			}
			reordered = append(reordered, p)
			delete(byID, id)
		}
		q.Items = reordered
	case "pause":
		q.Paused = true
	case "resume":
		if s.driver == nil || s.info.State == "disconnected" || s.info.State == "closed" {
			return PromptQueue{}, ErrUnavailable
		}
		q.Paused = false
	default:
		return PromptQueue{}, errors.New("unknown queue operation")
	}
	if err := s.writeQueueLocked(q); err != nil {
		return PromptQueue{}, err
	}
	b.drainQueueLocked(s)
	return cloneQueue(s.queue), nil
}
func (b *Broker) drainQueue(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	s := b.sessions[id]
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b.drainQueueLocked(s)
}
func (b *Broker) drainQueueLocked(s *liveSession) {
	if s.queue.Paused || len(s.queue.Items) == 0 || s.current != "" || s.info.State != "ready" || s.info.Archived || s.stopping || s.driver == nil {
		return
	}
	p := s.queue.Items[0]
	if _, err := b.startPromptLocked(s, p.ID, p.Text, p.RequestOptions, p.Options); err != nil {
		s.pauseQueueLocked()
		return
	}
	// startPromptLocked durably submitted the user message and removed it from
	// the in-memory queue through applyQueueEventLocked.
	_ = s.writeQueueLocked(cloneQueue(s.queue))
}
