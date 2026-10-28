package agentruntime

import (
	"context"
	"sort"
	"strings"
	"time"
)

// Current projection is limited to the latest turn and is derived as its
// events are accepted. No historical event is read to build it. The journal
// remains the authority; an oversized projection fails explicitly and its
// complete records remain available through the history API.
const (
	maxCurrentItems = 256
	maxCurrentBytes = 2 << 20
	maxCurrentText  = 256 << 10
	seedLeaseTime   = 15 * time.Second
)

type currentItem struct {
	event     Event
	text      strings.Builder
	baseBytes int
}
type currentTurnView struct {
	turn        string
	start       uint64
	items       []*currentItem
	byID        map[string]*currentItem
	bytes       int
	unavailable bool
	result      *Event
	resultBytes int
}

type CurrentInput struct {
	Request   Request          `json:"request"`
	Submitted bool             `json:"submitted"`
	Decision  *DecisionReceipt `json:"decision,omitempty"`
}
type CurrentSeed struct {
	Session       Session        `json:"session"`
	ActiveTurnID  string         `json:"activeTurnId,omitempty"`
	Items         []Event        `json:"items"`
	Inputs        []CurrentInput `json:"inputs"`
	Queue         *PromptQueue   `json:"queue,omitempty"`
	Result        *Event         `json:"result,omitempty"`
	HistoryBefore uint64         `json:"historyBefore,omitempty"`
}

func (s *liveSession) applyCurrentLocked(e Event, encodedBytes int) {
	if e.Kind == "item.snapshot" && e.Role == "user" && e.ItemID == "user" && e.Status == "submitted" {
		s.turnCount++
		s.currentView = &currentTurnView{turn: e.TurnID, start: e.Seq, byID: map[string]*currentItem{}}
	}
	view := s.currentView
	if view == nil || view.turn != e.TurnID || view.unavailable {
		return
	}
	if e.Kind == "turn.end" {
		result := e
		view.result = &result
		view.resultBytes = encodedBytes
		return
	}
	if e.Kind != "item.delta" && e.Kind != "item.snapshot" && e.Kind != "item.patch" {
		return
	}
	item := view.byID[e.ItemID]
	detailsChanged := false
	if item != nil && item.event.Details != nil && e.Details != nil {
		oldType, oldOK := item.event.Details["type"].(string)
		newType, newOK := e.Details["type"].(string)
		detailsChanged = !oldOK || !newOK || oldType != newType
	}
	if item == nil {
		if len(view.items) >= maxCurrentItems {
			view.unavailable = true
			view.items, view.byID = nil, nil
			return
		}
		item = &currentItem{}
		view.items = append(view.items, item)
		view.byID[e.ItemID] = item
	} else if item.event.Role != e.Role || (item.event.AgentThreadID != "" && e.AgentThreadID != "" && item.event.AgentThreadID != e.AgentThreadID) || (item.event.AgentTurnID != "" && e.AgentTurnID != "" && item.event.AgentTurnID != e.AgentTurnID) || detailsChanged {
		view.unavailable = true
		view.items, view.byID = nil, nil
		return
	}
	oldText := item.text.Len()
	newText := oldText
	if e.Kind == "item.snapshot" {
		newText = len(e.Text)
	}
	if e.Kind == "item.delta" {
		newText += len(e.Text)
	}
	// Six bytes per text byte is a conservative bound for JSON escaping. The
	// most recent event's encoded metadata also accounts for structured tools.
	base := encodedBytes - len(e.Text)
	if base < item.baseBytes {
		base = item.baseBytes
	}
	projected := view.bytes - item.baseBytes - 6*oldText + base + 6*newText
	if newText > maxCurrentText || projected > maxCurrentBytes {
		view.unavailable = true
		view.items, view.byID = nil, nil
		return
	}
	if e.Kind == "item.snapshot" {
		item.text.Reset()
	}
	if e.Kind != "item.patch" {
		_, _ = item.text.WriteString(e.Text)
	}
	if item.event.Seq == 0 {
		item.event = e
	} else {
		item.event.Seq = e.Seq
		if e.Title != "" {
			item.event.Title = e.Title
		}
		if e.Status != "" {
			item.event.Status = e.Status
		}
		if e.SourceMethod != "" {
			item.event.SourceMethod = e.SourceMethod
		}
		if e.AgentThreadID != "" {
			item.event.AgentThreadID = e.AgentThreadID
		}
		if e.AgentTurnID != "" {
			item.event.AgentTurnID = e.AgentTurnID
		}
		if e.Details != nil {
			item.event.Details = e.Details
		}
	}
	item.event.Text = ""
	item.baseBytes = base
	view.bytes = projected
}

func (b *Broker) Current(ctx context.Context, id string) (CurrentSeed, error) {
	if err := ctx.Err(); err != nil {
		return CurrentSeed{}, err
	}
	s, release, err := b.retainReader(id, false)
	if err != nil {
		return CurrentSeed{}, err
	}
	defer release()
	s.mu.Lock()
	defer s.mu.Unlock()
	seed := CurrentSeed{Session: s.info, ActiveTurnID: s.current, Items: []Event{}, Inputs: []CurrentInput{}}
	// Charge the complete session envelope, queue, current items and all
	// pending requests before the HTTP layer encodes the selected seed.
	budget := 64<<10 + s.queueBytes
	if s.ended {
		seed.ActiveTurnID = ""
	}
	if s.queueSeq > 0 {
		queue := cloneQueue(s.queue)
		seed.Queue = &queue
	}
	view := s.currentView
	if view == nil {
		if s.current != "" {
			return CurrentSeed{}, ErrUnavailable
		}
		if s.turnCount > 0 {
			seed.HistoryBefore = s.info.LastSeq + 1
		}
	} else {
		if view.unavailable {
			return CurrentSeed{}, exhausted("current turn exceeds the selected-view capacity; inspect its history explicitly")
		}
		budget += view.bytes + view.resultBytes
		if s.turnCount > 1 {
			seed.HistoryBefore = view.start
		}
		for _, item := range view.items {
			e := item.event
			e.Kind, e.Text, e.RuntimeID = "item.snapshot", item.text.String(), ""
			seed.Items = append(seed.Items, e)
		}
		seed.Result = view.result
	}
	for _, p := range s.activeInputs {
		if p.active && !s.ended && s.info.State != "cancelling" {
			budget += p.requestBytes + p.receiptBytes + 256
			seed.Inputs = append(seed.Inputs, CurrentInput{Request: p.request, Submitted: p.answer != nil, Decision: p.receipt})
		}
	}
	if budget > maxCurrentBytes {
		return CurrentSeed{}, exhausted("current turn and pending controls exceed the selected-view capacity; inspect history explicitly")
	}
	sort.Slice(seed.Inputs, func(i, j int) bool { return seed.Inputs[i].Request.ID < seed.Inputs[j].Request.ID })
	// Protect the bounded handoff until its subscriber attaches. No timer or
	// background job is created; a later lifecycle event may retire an expired
	// handoff if no subscriber arrives.
	s.seedLeaseUntil = time.Now().Add(seedLeaseTime)
	return seed, nil
}
