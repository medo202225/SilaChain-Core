package storage

import (
	"encoding/json"
	"os"

	"silachain/internal/validator"
)

type ValidatorStore struct {
	db *DB
}

func NewValidatorStore(db *DB) *ValidatorStore {
	return &ValidatorStore{db: db}
}

func (s *ValidatorStore) path() string {
	return join(s.db.BasePath, "validators.json")
}

func (s *ValidatorStore) Save(members []validator.Member) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(members, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *ValidatorStore) Load() ([]validator.Member, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return []validator.Member{}, nil
		}
		return nil, err
	}

	var out []validator.Member
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
