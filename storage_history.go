package agentruntime

import "context"

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

// replayedTurn reports whether the conversation already accepted the prompt
// with the identity turn: from memory while it is live, from storage otherwise
// (history is not retained once a conversation is idle). A prompt accepted with
// other content is ErrConflict. A replay returns only after the original
// message is durable. No lock is held while storage is read.
func (s *liveSession) replayedTurn(turn, fingerprint string) (bool, error) {
	s.mu.Lock()
	known, ok := s.turns[turn]
	s.mu.Unlock()
	if ok {
		if known.fingerprint != fingerprint {
			return false, ErrConflict
		}
		return true, s.waitDurable(context.Background(), known.seq)
	}
	stored, found, err := s.store.prompt(s.id, turn)
	if err != nil || !found {
		return false, err
	}
	if stored != fingerprint {
		return false, ErrConflict
	}
	return true, nil
}
