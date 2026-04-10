package storage

import (
	"encoding/json"
	"os"

	"silachain/internal/staking"
)

type StakingStore struct {
	db *DB
}

func NewStakingStore(db *DB) *StakingStore {
	return &StakingStore{db: db}
}

func (s *StakingStore) path() string {
	return join(s.db.BasePath, "staking.json")
}

func (s *StakingStore) Save(entries []staking.Entry) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *StakingStore) Load() ([]staking.Entry, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return []staking.Entry{}, nil
		}
		return nil, err
	}

	var out []staking.Entry
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
