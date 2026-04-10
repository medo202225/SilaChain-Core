package validatorclient

// CANONICAL OWNERSHIP: validator client package for keystores, duties, slashing protection, signing, and validator service runtime.
// Planned final architectural name is validatorclient after dependency cleanup.

type ValidatorRecord struct {
	PublicKey           string `json:"public_key"`
	WithdrawalPublicKey string `json:"withdrawal_public_key"`
	EffectiveBalance    uint64 `json:"effective_balance"`
	Slashed             bool   `json:"slashed"`
	ActivationEpoch     uint64 `json:"activation_epoch"`
	ExitEpoch           uint64 `json:"exit_epoch"`
	WithdrawableEpoch   uint64 `json:"withdrawable_epoch"`
}

type ValidatorSet struct {
	Validators []ValidatorRecord
}

func (s *ValidatorSet) All() []ValidatorRecord {
	if s == nil {
		return nil
	}
	return s.Validators
}
