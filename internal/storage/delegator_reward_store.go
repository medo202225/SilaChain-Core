package storage

import (
	"encoding/json"
	"os"

	"silachain/internal/staking"
)

type DelegatorRewardStore struct {
	db *DB
}

func NewDelegatorRewardStore(db *DB) *DelegatorRewardStore {
	return &DelegatorRewardStore{db: db}
}

func (s *DelegatorRewardStore) path() string {
	return join(s.db.BasePath, "delegator_rewards.json")
}

func (s *DelegatorRewardStore) Save(items []staking.DelegatorReward) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *DelegatorRewardStore) Load() ([]staking.DelegatorReward, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return []staking.DelegatorReward{}, nil
		}
		return nil, err
	}

	var out []staking.DelegatorReward
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
