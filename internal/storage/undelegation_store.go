package storage

import (
	"encoding/json"
	"os"

	"silachain/internal/staking"
)

type UndelegationStore struct {
	db *DB
}

func NewUndelegationStore(db *DB) *UndelegationStore {
	return &UndelegationStore{db: db}
}

func (s *UndelegationStore) path() string {
	return join(s.db.BasePath, "undelegations.json")
}

func (s *UndelegationStore) Save(items []staking.Undelegation) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *UndelegationStore) Load() ([]staking.Undelegation, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return []staking.Undelegation{}, nil
		}
		return nil, err
	}

	var out []staking.Undelegation
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
