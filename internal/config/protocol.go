package config

type ProtocolConfig struct {
	BlockReward            uint64 `json:"block_reward"`
	UnbondingDelay         uint64 `json:"unbonding_delay"`
	MinValidatorStake      uint64 `json:"min_validator_stake"`
	ValidatorCommissionBps uint64 `json:"validator_commission_bps"`
}

func DefaultProtocolConfig() ProtocolConfig {
	return ProtocolConfig{
		BlockReward:            10,
		UnbondingDelay:         3,
		MinValidatorStake:      1,
		ValidatorCommissionBps: 1000,
	}
}
