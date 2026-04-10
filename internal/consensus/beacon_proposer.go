package consensus

func (s *BeaconStateV1) ProposerForCurrentSlot() *ValidatorRecord {
	active := s.ActiveValidators()
	if len(active) == 0 {
		return nil
	}

	idx := int(s.Slot % uint64(len(active)))
	v := active[idx]
	return &v
}
