package agentruntime

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

func (s *liveSession) readEventsLocked(after, end uint64) ([]Event, error) {
	events, err := s.store.readEvents(s.info.ID, after, end)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.Provider != s.info.Provider {
			return nil, ErrConflict
		}
	}
	return events, nil
}

func (s *liveSession) findTurnLocked(turn string) (string, bool, error) {
	return s.store.prompt(s.info.ID, turn)
}
