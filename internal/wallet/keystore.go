package wallet

import (
	"encoding/json"
	"os"
)

func SaveToFile(path string, w *Wallet) error {
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func LoadFromFile(path string) (*Wallet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var w Wallet
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}

	return &w, nil
}
