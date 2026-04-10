package consensus

func (s *BeaconStateV1) slotIndexInEpoch(slotsPerEpoch uint64) uint64 {
	if slotsPerEpoch == 0 {
		slotsPerEpoch = 32
	}
	return s.Slot % slotsPerEpoch
}

func (s *BeaconStateV1) CommitteeCount(slotsPerEpoch uint64) int {
	active := s.ActiveValidators()
	if len(active) == 0 {
		return 0
	}

	if slotsPerEpoch == 0 {
		slotsPerEpoch = 32
	}

	count := len(active)
	if count > int(slotsPerEpoch) {
		count = int(slotsPerEpoch)
	}
	if count < 1 {
		count = 1
	}
	return count
}

func (s *BeaconStateV1) CommitteeIndexForCurrentSlot(slotsPerEpoch uint64) int {
	count := s.CommitteeCount(slotsPerEpoch)
	if count == 0 {
		return 0
	}
	return int(s.slotIndexInEpoch(slotsPerEpoch) % uint64(count))
}

func (s *BeaconStateV1) shuffledActiveValidators() []ValidatorRecord {
	active := s.ActiveValidators()
	if len(active) <= 1 {
		return active
	}

	out := make([]ValidatorRecord, len(active))
	copy(out, active)

	shift := 0
	if len(out) > 0 {
		shift = int(s.Epoch % uint64(len(out)))
	}
	if shift == 0 {
		return out
	}

	rotated := make([]ValidatorRecord, 0, len(out))
	rotated = append(rotated, out[shift:]...)
	rotated = append(rotated, out[:shift]...)
	return rotated
}

func (s *BeaconStateV1) AttestersForCurrentSlot() []ValidatorRecord {
	active := s.shuffledActiveValidators()
	if len(active) == 0 {
		return nil
	}

	slotsPerEpoch := uint64(32)
	committeeCount := s.CommitteeCount(slotsPerEpoch)
	if committeeCount == 0 {
		return nil
	}

	committeeIndex := s.CommitteeIndexForCurrentSlot(slotsPerEpoch)

	baseSize := len(active) / committeeCount
	remainder := len(active) % committeeCount

	size := baseSize
	if committeeIndex < remainder {
		size++
	}
	if size <= 0 {
		size = 1
	}

	start := 0
	for i := 0; i < committeeIndex; i++ {
		chunk := baseSize
		if i < remainder {
			chunk++
		}
		start += chunk
	}

	end := start + size
	if start >= len(active) {
		start = len(active) - 1
	}
	if end > len(active) {
		end = len(active)
	}

	out := make([]ValidatorRecord, 0, end-start)
	out = append(out, active[start:end]...)
	return out
}

func (s *BeaconStateV1) HasAttester(publicKey string) bool {
	attesters := s.AttestersForCurrentSlot()
	for _, v := range attesters {
		if v.PublicKey == publicKey {
			return true
		}
	}
	return false
}

func (s *BeaconStateV1) AttesterByPublicKey(publicKey string) *ValidatorRecord {
	attesters := s.AttestersForCurrentSlot()
	for _, v := range attesters {
		if v.PublicKey == publicKey {
			vv := v
			return &vv
		}
	}
	return nil
}
