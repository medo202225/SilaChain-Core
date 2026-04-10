package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ChainMetadata struct {
	NextValidatorIndex int    `json:"next_validator_index"`
	Epoch              uint64 `json:"epoch"`
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func join(base string, parts ...string) string {
	all := append([]string{base}, parts...)
	return filepath.Join(all...)
}

type MetadataStore struct {
	db *DB
}

func NewMetadataStore(db *DB) *MetadataStore {
	return &MetadataStore{db: db}
}

func (s *MetadataStore) path() string {
	return join(s.db.BasePath, "metadata.json")
}

func (s *MetadataStore) Save(meta ChainMetadata) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *MetadataStore) Load() (ChainMetadata, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return ChainMetadata{}, nil
		}
		return ChainMetadata{}, err
	}

	var out ChainMetadata
	if err := json.Unmarshal(data, &out); err != nil {
		return ChainMetadata{}, err
	}
	return out, nil
}
