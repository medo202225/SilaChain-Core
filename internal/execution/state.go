package execution

import (
	"fmt"
	"sync"
)

type ImportedBlock struct {
	Number     uint64
	Hash       string
	ParentHash string
	Timestamp  uint64
	TxHashes   []string
}

type PendingTx struct {
	Hash  string
	From  string
	To    string
	Value string
	Nonce uint64
	Data  string
}

type State struct {
	mu          sync.RWMutex
	genesisHash string
	headNumber  uint64
	headHash    string
	blocks      map[uint64]ImportedBlock
	pendingTxs  map[string]PendingTx
}

func NewState(genesisHash string) *State {
	s := &State{
		genesisHash: genesisHash,
		headNumber:  0,
		headHash:    genesisHash,
		blocks:      make(map[uint64]ImportedBlock),
		pendingTxs:  make(map[string]PendingTx),
	}

	s.blocks[0] = ImportedBlock{
		Number:     0,
		Hash:       genesisHash,
		ParentHash: "",
		Timestamp:  0,
		TxHashes:   nil,
	}
	return s
}

func (s *State) ImportBlock(block ImportedBlock) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if block.Hash == "" {
		return fmt.Errorf("execution state: empty block hash")
	}

	if existing, ok := s.blocks[block.Number]; ok {
		if existing.Hash == block.Hash {
			return nil
		}
		return fmt.Errorf("execution state: conflicting block at height %d", block.Number)
	}

	if block.Number == 0 {
		if block.Hash != s.genesisHash {
			return fmt.Errorf("execution state: invalid genesis hash")
		}
		s.blocks[0] = block
		s.headNumber = 0
		s.headHash = block.Hash
		return nil
	}

	parent, ok := s.blocks[block.Number-1]
	if !ok {
		return fmt.Errorf("execution state: missing parent block %d", block.Number-1)
	}
	if block.ParentHash != parent.Hash {
		return fmt.Errorf("execution state: parent hash mismatch for block %d", block.Number)
	}

	s.blocks[block.Number] = block
	if block.Number >= s.headNumber {
		s.headNumber = block.Number
		s.headHash = block.Hash
	}
	return nil
}

func (s *State) AddPendingTx(tx PendingTx) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tx.Hash == "" {
		return false
	}
	if _, ok := s.pendingTxs[tx.Hash]; ok {
		return false
	}
	s.pendingTxs[tx.Hash] = tx
	return true
}

func (s *State) HasPendingTx(hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.pendingTxs[hash]
	return ok
}

func (s *State) PendingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pendingTxs)
}

func (s *State) HeadNumber() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.headNumber
}

func (s *State) HeadHash() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.headHash
}
