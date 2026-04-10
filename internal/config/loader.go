package config

import (
	"encoding/json"
	"os"
)

func LoadExecutionNodeConfig(path string) (*ExecutionNodeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ExecutionNodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	cfg.Normalize()
	return &cfg, nil
}

func LoadValidatorClientConfig(path string) (*ValidatorClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ValidatorClientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	cfg.Normalize()
	return &cfg, nil
}

func LoadConsensusClientConfig(path string) (*ConsensusClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ConsensusClientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	cfg.Normalize()
	return &cfg, nil
}
