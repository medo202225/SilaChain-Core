package state

type StateMetrics struct {
	AccountCount      int
	DirtyAccountCount int
	JournalLength     int
	RevisionCount     int
	LogCount          int
	Refund            uint64
}

func (s *StateDB) Metrics() StateMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := StateMetrics{
		AccountCount:      len(s.stateObjects),
		DirtyAccountCount: len(s.stateObjectsDirty),
		RevisionCount:     len(s.revisions),
		LogCount:          len(s.logs),
		Refund:            s.refund,
	}
	if s.journal != nil {
		metrics.JournalLength = s.journal.length()
	}
	return metrics
}
