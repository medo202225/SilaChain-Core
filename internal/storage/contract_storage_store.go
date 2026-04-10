package storage

import (
	"encoding/json"
	"os"

	"silachain/pkg/types"
)

type ContractStorageStore struct {
	db *DB
}

func NewContractStorageStore(db *DB) *ContractStorageStore {
	return &ContractStorageStore{db: db}
}

func (s *ContractStorageStore) path() string {
	return join(s.db.BasePath, "contract_storage.json")
}

func (s *ContractStorageStore) Save(data map[types.Address]map[string]string) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), raw, 0o600)
}

func (s *ContractStorageStore) Load() (map[types.Address]map[string]string, error) {
	raw, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return map[types.Address]map[string]string{}, nil
		}
		return nil, err
	}

	var out map[types.Address]map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
