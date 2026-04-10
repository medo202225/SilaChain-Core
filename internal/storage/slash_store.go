package storage

import (
	"encoding/json"
	"os"

	"silachain/internal/staking"
)

type SlashStore struct {
	db *DB
}

func NewSlashStore(db *DB) *SlashStore {
	return &SlashStore{db: db}
}

func (s *SlashStore) path() string {
	return join(s.db.BasePath, "slashes.json")
}

func (s *SlashStore) Save(items []staking.Slash) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *SlashStore) Load() ([]staking.Slash, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return []staking.Slash{}, nil
		}
		return nil, err
	}

	var out []staking.Slash
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
