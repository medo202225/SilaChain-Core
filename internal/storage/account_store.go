package storage

// CANONICAL OWNERSHIP: persistence layer for chain data and domain stores.

import (
	"encoding/json"
	"os"

	"silachain/internal/accounts"
)

type AccountStore struct {
	db *DB
}

func NewAccountStore(db *DB) *AccountStore {
	return &AccountStore{db: db}
}

func (s *AccountStore) path() string {
	return join(s.db.BasePath, "accounts.json")
}

func (s *AccountStore) Save(manager *accounts.Manager) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(manager.All(), "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *AccountStore) Load() (map[string]*accounts.Account, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]*accounts.Account{}, nil
		}
		return nil, err
	}

	var out map[string]*accounts.Account
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
