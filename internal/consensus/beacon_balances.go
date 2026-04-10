package consensus

func (s *BeaconStateV1) TotalActiveBalance() uint64 {
	active := s.ActiveValidators()
	var total uint64
	for _, v := range active {
		total += v.EffectiveBalance
	}
	return total
}

func (s *BeaconStateV1) ActiveValidatorCount() int {
	return len(s.ActiveValidators())
}
