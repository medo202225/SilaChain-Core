package storage

import (
	"encoding/json"
	"os"
)

type TxLocation struct {
	BlockHeight uint64 `json:"block_height"`
	TxIndex     int    `json:"tx_index"`
}

type TxStore struct {
	db *DB
}

func NewTxStore(db *DB) *TxStore {
	return &TxStore{db: db}
}

func (s *TxStore) path() string {
	return join(s.db.BasePath, "tx_index.json")
}

func (s *TxStore) Save(index map[string]TxLocation) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *TxStore) Load() (map[string]TxLocation, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]TxLocation{}, nil
		}
		return nil, err
	}

	var out map[string]TxLocation
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
