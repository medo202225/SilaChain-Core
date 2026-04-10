package consensuslegacy

import (
	"encoding/json"
	"os"
	"time"
)

type ProtocolConfig struct {
	BlockReward            uint64 `json:"block_reward"`
	UnbondingDelay         uint64 `json:"unbonding_delay"`
	MinValidatorStake      uint64 `json:"min_validator_stake"`
	ValidatorCommissionBps uint64 `json:"validator_commission_bps"`
	GenesisTimeUnix        int64  `json:"genesis_time_unix"`
	SlotDurationSeconds    uint64 `json:"slot_duration_seconds"`
	SlotsPerEpoch          uint64 `json:"slots_per_epoch"`
}

func LoadProtocolConfig(path string) (ProtocolConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ProtocolConfig{}, err
	}

	var cfg ProtocolConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return ProtocolConfig{}, err
	}

	if cfg.SlotDurationSeconds == 0 {
		cfg.SlotDurationSeconds = 12
	}
	if cfg.SlotsPerEpoch == 0 {
		cfg.SlotsPerEpoch = 32
	}

	return cfg, nil
}

func (p ProtocolConfig) ToConsensusConfig() Config {
	cfg := DefaultConfig()
	cfg.GenesisTimeUnix = p.GenesisTimeUnix
	cfg.SlotDuration = time.Duration(p.SlotDurationSeconds) * time.Second
	cfg.SlotsPerEpoch = p.SlotsPerEpoch
	return cfg
}
