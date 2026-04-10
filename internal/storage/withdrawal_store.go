package storage

import (
	"encoding/json"
	"os"

	"silachain/internal/staking"
)

type WithdrawalStore struct {
	db *DB
}

func NewWithdrawalStore(db *DB) *WithdrawalStore {
	return &WithdrawalStore{db: db}
}

func (s *WithdrawalStore) path() string {
	return join(s.db.BasePath, "reward_withdrawals.json")
}

func (s *WithdrawalStore) Save(items []staking.Withdrawal) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *WithdrawalStore) Load() ([]staking.Withdrawal, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return []staking.Withdrawal{}, nil
		}
		return nil, err
	}

	var out []staking.Withdrawal
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
