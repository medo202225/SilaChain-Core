package app

import (
	"encoding/json"
	"os"
)

func LoadPeersFile(path string, selfURL string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Peers []string `json:"peers"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}

	return UniquePeers(payload.Peers, selfURL), nil
}

func SavePeersFile(path string, peers []string, selfURL string) error {
	payload := struct {
		Peers []string `json:"peers"`
	}{
		Peers: UniquePeers(peers, selfURL),
	}

	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, raw, 0o644)
}
