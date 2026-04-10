package storage

// CANONICAL OWNERSHIP: persistence layer for chain data and domain stores.

import (
	"encoding/json"
	"os"

	"silachain/internal/core/types"
)

type BlockStore struct {
	db *DB
}

func NewBlockStore(db *DB) *BlockStore {
	return &BlockStore{db: db}
}

func (s *BlockStore) path() string {
	return join(s.db.BasePath, "blocks.json")
}

func (s *BlockStore) Save(blocks []*types.Block) error {
	if err := ensureDir(s.db.BasePath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(blocks, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path(), data, 0o600)
}

func (s *BlockStore) Load() ([]*types.Block, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return []*types.Block{}, nil
		}
		return nil, err
	}

	var blocks []*types.Block
	if err := json.Unmarshal(data, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}
