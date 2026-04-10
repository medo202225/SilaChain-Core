package storage

import (
	"encoding/json"
	"os"

	"silachain/internal/staking"
)

type JailStore struct {
	db *DB
}

func NewJailStore(db *DB) *JailStore {
	return &JailStore{db: db}
}

func (s *JailStore) path() string {
	return join(s.db.BasePath, "jails.json")
}

func (s *JailStore) Save(items []staking.Jail) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *JailStore) Load() ([]staking.Jail, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return []staking.Jail{}, nil
		}
		return nil, err
	}

	var out []staking.Jail
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
