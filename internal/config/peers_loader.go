package config

import (
	"encoding/json"
	"os"
)

func LoadPeersConfig(path string) (PeersConfig, error) {
	cfg := PeersConfig{
		Peers: []string{},
	}

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

	return cfg, nil
}
