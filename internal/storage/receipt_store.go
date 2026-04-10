package storage

import (
	"encoding/json"
	"os"

	"silachain/pkg/types"
)

type ReceiptStore struct {
	db *DB
}

func NewReceiptStore(db *DB) *ReceiptStore {
	return &ReceiptStore{db: db}
}

func (s *ReceiptStore) path() string {
	return join(s.db.BasePath, "receipts.json")
}

func (s *ReceiptStore) Save(receipts map[string]types.Receipt) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(receipts, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *ReceiptStore) Load() (map[string]types.Receipt, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]types.Receipt{}, nil
		}
		return nil, err
	}

	var out map[string]types.Receipt
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
