package consensus

func (s *BeaconStateV1) IsValidatorActive(v ValidatorRecord) bool {
	if v.Slashed {
		return false
	}
	if v.ActivationEpoch == farFutureEpoch {
		return false
	}
	if s.Epoch < v.ActivationEpoch {
		return false
	}
	if v.ExitEpoch != farFutureEpoch && s.Epoch >= v.ExitEpoch {
		return false
	}
	return true
}

func (s *BeaconStateV1) IsValidatorEligibleForActivation(v ValidatorRecord) bool {
	if v.Slashed {
		return false
	}
	if v.EffectiveBalance < maxEffectiveBalance {
		return false
	}
	if v.ActivationEligibilityEpoch == farFutureEpoch {
		return false
	}
	if v.ActivationEpoch != farFutureEpoch {
		return false
	}
	return true
}

func (s *BeaconStateV1) IsValidatorExited(v ValidatorRecord) bool {
	if v.ExitEpoch == farFutureEpoch {
		return false
	}
	return s.Epoch >= v.ExitEpoch
}

func (s *BeaconStateV1) IsValidatorWithdrawable(v ValidatorRecord) bool {
	if v.WithdrawableEpoch == farFutureEpoch {
		return false
	}
	return s.Epoch >= v.WithdrawableEpoch
}

func (s *BeaconStateV1) CanValidatorVoluntarilyExit(index int, exitEpoch uint64) bool {
	if s == nil || index < 0 || index >= len(s.Validators) {
		return false
	}

	v := s.Validators[index]
	if v.Slashed {
		return false
	}
	if v.ActivationEpoch == farFutureEpoch {
		return false
	}
	if s.Epoch < v.ActivationEpoch {
		return false
	}
	if v.ExitEpoch != farFutureEpoch {
		return false
	}
	if exitEpoch > s.Epoch {
		return false
	}

	return true
}

func (s *BeaconStateV1) CanSlashValidator(index int) bool {
	if s == nil || index < 0 || index >= len(s.Validators) {
		return false
	}

	v := s.Validators[index]
	if v.Slashed {
		return false
	}
	if index >= len(s.Balances) {
		return false
	}
	if s.Balances[index] == 0 {
		return false
	}

	return true
}

func (s *BeaconStateV1) ActiveValidators() []ValidatorRecord {
	if s == nil {
		return nil
	}

	out := make([]ValidatorRecord, 0, len(s.Validators))
	for _, v := range s.Validators {
		if !s.IsValidatorActive(v) {
			continue
		}
		out = append(out, v)
	}
	return out
}

func (s *BeaconStateV1) PendingActivationValidators() []ValidatorRecord {
	if s == nil {
		return nil
	}

	out := make([]ValidatorRecord, 0, len(s.Validators))
	for _, v := range s.Validators {
		if !s.IsValidatorEligibleForActivation(v) {
			continue
		}
		out = append(out, v)
	}
	return out
}

func (s *BeaconStateV1) PendingActivationIndices() []int {
	if s == nil {
		return nil
	}

	out := make([]int, 0, len(s.Validators))
	for i, v := range s.Validators {
		if !s.IsValidatorEligibleForActivation(v) {
			continue
		}
		out = append(out, i)
	}
	return out
}

func (s *BeaconStateV1) ExitedValidators() []ValidatorRecord {
	if s == nil {
		return nil
	}

	out := make([]ValidatorRecord, 0, len(s.Validators))
	for _, v := range s.Validators {
		if !s.IsValidatorExited(v) {
			continue
		}
		out = append(out, v)
	}
	return out
}

func (s *BeaconStateV1) ExitedButNotWithdrawableValidators() []ValidatorRecord {
	if s == nil {
		return nil
	}

	out := make([]ValidatorRecord, 0, len(s.Validators))
	for _, v := range s.Validators {
		if !s.IsValidatorExited(v) {
			continue
		}
		if s.IsValidatorWithdrawable(v) {
			continue
		}
		out = append(out, v)
	}
	return out
}

func (s *BeaconStateV1) WithdrawableValidators() []ValidatorRecord {
	if s == nil {
		return nil
	}

	out := make([]ValidatorRecord, 0, len(s.Validators))
	for _, v := range s.Validators {
		if !s.IsValidatorWithdrawable(v) {
			continue
		}
		out = append(out, v)
	}
	return out
}

func (s *BeaconStateV1) SlashedValidators() []ValidatorRecord {
	if s == nil {
		return nil
	}

	out := make([]ValidatorRecord, 0, len(s.Validators))
	for _, v := range s.Validators {
		if v.Slashed {
			out = append(out, v)
		}
	}
	return out
}
