package consensus

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	maxEffectiveBalance       uint64 = 32000000000
	effectiveBalanceIncrement uint64 = 1000000000
)

type beaconValidatorJSON struct {
	PublicKey                  string `json:"public_key"`
	WithdrawalPublicKey        string `json:"withdrawal_public_key"`
	WithdrawalCredentials      string `json:"withdrawal_credentials"`
	EffectiveBalance           uint64 `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch uint64 `json:"activation_eligibility_epoch"`
	ActivationEpoch            uint64 `json:"activation_epoch"`
	ExitEpoch                  uint64 `json:"exit_epoch"`
	WithdrawableEpoch          uint64 `json:"withdrawable_epoch"`
}

func normalizeEffectiveBalance(v uint64) uint64 {
	if v == 0 {
		return 0
	}
	if v > maxEffectiveBalance {
		v = maxEffectiveBalance
	}
	return v - (v % effectiveBalanceIncrement)
}

func LoadBeaconValidatorsFromFile(path string) ([]ValidatorRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read validators file failed: %w", err)
	}

	var items []beaconValidatorJSON
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("parse validators file failed: %w", err)
	}

	out := make([]ValidatorRecord, 0, len(items))
	for _, v := range items {
		withdrawalCredentials := v.WithdrawalCredentials
		if withdrawalCredentials == "" {
			withdrawalCredentials = v.WithdrawalPublicKey
		}

		exitEpoch := v.ExitEpoch
		if exitEpoch == 0 {
			exitEpoch = farFutureEpoch
		}

		withdrawableEpoch := v.WithdrawableEpoch
		if withdrawableEpoch == 0 {
			withdrawableEpoch = farFutureEpoch
		}

		out = append(out, ValidatorRecord{
			PublicKey:                  v.PublicKey,
			WithdrawalCredentials:      withdrawalCredentials,
			EffectiveBalance:           normalizeEffectiveBalance(v.EffectiveBalance),
			Slashed:                    v.Slashed,
			ActivationEligibilityEpoch: v.ActivationEligibilityEpoch,
			ActivationEpoch:            v.ActivationEpoch,
			ExitEpoch:                  exitEpoch,
			WithdrawableEpoch:          withdrawableEpoch,
		})
	}

	return out, nil
}
