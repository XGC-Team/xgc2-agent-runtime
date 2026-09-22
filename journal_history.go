package agentruntime

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
)

func (s *liveSession) applyMetadataLocked(event Event) {
	s.applyQueueEventLocked(event)
	if event.Kind == "item.snapshot" && event.Role == "user" && event.ItemID == "user" && event.Status == "submitted" {
		options := optionsFromDetails(event.Details)
		s.info.Options = mergeOptions(s.info.Options, options)
		s.profile.Defaults = mergeOptions(s.profile.Defaults, options)
	}
	if event.Kind == "session.metadata" && event.Metadata != nil {
		s.info.Title = event.Metadata.Title
		s.info.Archived = event.Metadata.Archived
		s.info.MetadataRevision = event.Metadata.MetadataRevision
	}
	if event.Kind == "session.runtime" {
		s.info.RuntimeID = event.RuntimeID
	}
}

// scanEventsLocked does not retain transcript content. The journal is bounded
// independently of how many historical conversations an operator retains.
func (s *liveSession) scanEventsLocked(visit func(Event) bool) error {
	file, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), MaxFrame)
	if !scanner.Scan() {
		return errors.New("journal metadata missing")
	}
	var seq uint64
	for scanner.Scan() {
		var event Event
		if json.Unmarshal(scanner.Bytes(), &event) != nil || event.SchemaVersion != Schema || event.SessionID != s.info.ID || event.Provider != s.info.Provider || event.Seq != seq+1 {
			return errors.New("journal event identity or sequence mismatch")
		}
		seq = event.Seq
		if !visit(event) {
			return nil
		}
	}
	if scanner.Err() != nil || seq != s.info.LastSeq {
		return errors.New("journal could not be read completely")
	}
	return nil
}

func (s *liveSession) readEventsLocked(after, end uint64) ([]Event, error) {
	result := []Event{}
	if after == end {
		return result, nil
	}
	err := s.scanEventsLocked(func(event Event) bool {
		if event.Seq > after && event.Seq <= end {
			result = append(result, event)
		}
		return event.Seq < end
	})
	if err == nil && uint64(len(result)) != end-after {
		err = errors.New("journal replay range is unavailable")
	}
	return result, err
}

func (s *liveSession) findTurnLocked(turn string) (string, bool, error) {
	if fingerprint, ok := s.turns[turn]; ok {
		return fingerprint, true, nil
	}
	if s.events != nil {
		return "", false, nil
	}
	var fingerprint string
	found := false
	err := s.scanEventsLocked(func(event Event) bool {
		if event.Kind == "item.snapshot" && event.Role == "user" && event.TurnID == turn {
			fingerprint = promptFingerprint(event.Text, optionsFromDetails(event.Details))
			found = true
			return false
		}
		return true
	})
	return fingerprint, found, err
}

func (s *liveSession) activateJournalLocked() error {
	if s.events != nil {
		return nil
	}
	events, err := s.readEventsLocked(0, s.info.LastSeq)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(s.path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	s.file = file
	s.events = events
	for _, event := range events {
		if event.Kind == "item.snapshot" && event.Role == "user" {
			s.turns[event.TurnID] = promptFingerprint(event.Text, optionsFromDetails(event.Details))
		}
	}
	return nil
}
