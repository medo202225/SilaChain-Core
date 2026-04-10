package config

import (
	"encoding/json"
	"os"
)

func LoadProtocolConfig(path string) (ProtocolConfig, error) {
	cfg := DefaultProtocolConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if cfg.BlockReward == 0 {
		cfg.BlockReward = 10
	}
	if cfg.UnbondingDelay == 0 {
		cfg.UnbondingDelay = 3
	}
	if cfg.MinValidatorStake == 0 {
		cfg.MinValidatorStake = 1
	}
	if cfg.ValidatorCommissionBps > 10000 {
		cfg.ValidatorCommissionBps = 10000
	}

	return cfg, nil
}
