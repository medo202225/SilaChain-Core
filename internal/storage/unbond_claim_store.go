package storage

import (
	"encoding/json"
	"os"

	"silachain/internal/staking"
)

type UnbondClaimStore struct {
	db *DB
}

func NewUnbondClaimStore(db *DB) *UnbondClaimStore {
	return &UnbondClaimStore{db: db}
}

func (s *UnbondClaimStore) path() string {
	return join(s.db.BasePath, "unbond_claims.json")
}

func (s *UnbondClaimStore) Save(items []staking.UnbondClaim) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *UnbondClaimStore) Load() ([]staking.UnbondClaim, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return []staking.UnbondClaim{}, nil
		}
		return nil, err
	}

	var out []staking.UnbondClaim
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
