package storage

import (
	"encoding/json"
	"os"

	"silachain/internal/staking"
)

type RewardStore struct {
	db *DB
}

func NewRewardStore(db *DB) *RewardStore {
	return &RewardStore{db: db}
}

func (s *RewardStore) path() string {
	return join(s.db.BasePath, "rewards.json")
}

func (s *RewardStore) Save(items []staking.Reward) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *RewardStore) Load() ([]staking.Reward, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return []staking.Reward{}, nil
		}
		return nil, err
	}

	var out []staking.Reward
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
