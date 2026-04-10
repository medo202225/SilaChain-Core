package storage

import (
	"encoding/json"
	"os"

	"silachain/internal/staking"
)

type DelegationStore struct {
	db *DB
}

func NewDelegationStore(db *DB) *DelegationStore {
	return &DelegationStore{db: db}
}

func (s *DelegationStore) path() string {
	return join(s.db.BasePath, "delegations.json")
}

func (s *DelegationStore) Save(items []staking.Delegation) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *DelegationStore) Load() ([]staking.Delegation, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return []staking.Delegation{}, nil
		}
		return nil, err
	}

	var out []staking.Delegation
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
