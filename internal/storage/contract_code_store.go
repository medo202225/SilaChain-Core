package storage

import (
	"encoding/json"
	"os"

	"silachain/pkg/types"
)

type ContractCodeStore struct {
	db *DB
}

func NewContractCodeStore(db *DB) *ContractCodeStore {
	return &ContractCodeStore{db: db}
}

func (s *ContractCodeStore) path() string {
	return join(s.db.BasePath, "contract_code.json")
}

func (s *ContractCodeStore) Save(code map[types.Address]string) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(code, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *ContractCodeStore) Load() (map[types.Address]string, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return map[types.Address]string{}, nil
		}
		return nil, err
	}

	var out map[types.Address]string
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
