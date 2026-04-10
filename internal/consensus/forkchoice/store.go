package forkchoice

import (
	"errors"
	"fmt"

	"silachain/internal/consensus/blockassembly"
)

var (
	ErrEmptyGenesisHash   = errors.New("forkchoice: empty genesis hash")
	ErrEmptyBlockHash     = errors.New("forkchoice: empty block hash")
	ErrEmptyParentHash    = errors.New("forkchoice: empty parent hash")
	ErrUnknownParent      = errors.New("forkchoice: unknown parent")
	ErrUnknownBlock       = errors.New("forkchoice: unknown block")
	ErrBlockNumberGap     = errors.New("forkchoice: invalid block number progression")
	ErrCanonicalHeadUnset = errors.New("forkchoice: canonical head unset")
)

type BlockRef struct {
	Number     uint64 `json:"number"`
	Hash       string `json:"hash"`
	ParentHash string `json:"parentHash"`
	StateRoot  string `json:"stateRoot"`
}

type ApplyResult struct {
	Accepted         bool     `json:"accepted"`
	CanonicalChanged bool     `json:"canonicalChanged"`
	CanonicalHead    BlockRef `json:"canonicalHead"`
	SafeHead         BlockRef `json:"safeHead"`
	FinalizedHead    BlockRef `json:"finalizedHead"`
}

type Store struct {
	blocks         map[string]BlockRef
	canonicalHead  BlockRef
	safeHead       BlockRef
	finalizedHead  BlockRef
	canonicalByNum map[uint64]string
	hasHead        bool
}

func New(genesis blockassembly.Head) (*Store, error) {
	if genesis.Hash == "" {
		return nil, ErrEmptyGenesisHash
	}

	genesisRef := BlockRef{
		Number:     genesis.Number,
		Hash:       genesis.Hash,
		ParentHash: "",
		StateRoot:  genesis.StateRoot,
	}

	return &Store{
		blocks: map[string]BlockRef{
			genesisRef.Hash: genesisRef,
		},
		canonicalHead: genesisRef,
		safeHead:      genesisRef,
		finalizedHead: genesisRef,
		canonicalByNum: map[uint64]string{
			genesisRef.Number: genesisRef.Hash,
		},
		hasHead: true,
	}, nil
}

func (s *Store) HasBlock(hash string) bool {
	if s == nil {
		return false
	}
	_, ok := s.blocks[hash]
	return ok
}

func (s *Store) GetBlock(hash string) (BlockRef, bool) {
	if s == nil {
		return BlockRef{}, false
	}
	block, ok := s.blocks[hash]
	return block, ok
}

func (s *Store) GetCanonicalBlockByNumber(number uint64) (BlockRef, bool) {
	if s == nil {
		return BlockRef{}, false
	}
	hash, ok := s.canonicalByNum[number]
	if !ok {
		return BlockRef{}, false
	}
	block, ok := s.blocks[hash]
	return block, ok
}

func (s *Store) CanonicalHead() (BlockRef, error) {
	if s == nil || !s.hasHead {
		return BlockRef{}, ErrCanonicalHeadUnset
	}
	return s.canonicalHead, nil
}

func (s *Store) SafeHead() (BlockRef, error) {
	if s == nil || !s.hasHead {
		return BlockRef{}, ErrCanonicalHeadUnset
	}
	return s.safeHead, nil
}

func (s *Store) FinalizedHead() (BlockRef, error) {
	if s == nil || !s.hasHead {
		return BlockRef{}, ErrCanonicalHeadUnset
	}
	return s.finalizedHead, nil
}

func (s *Store) CanonicalBlocks(limit int) ([]BlockRef, error) {
	if s == nil || !s.hasHead {
		return nil, ErrCanonicalHeadUnset
	}
	if limit <= 0 {
		limit = 10
	}

	blocks := make([]BlockRef, 0, limit)
	number := s.canonicalHead.Number

	for len(blocks) < limit {
		hash, ok := s.canonicalByNum[number]
		if !ok {
			break
		}
		block, ok := s.blocks[hash]
		if !ok {
			break
		}
		blocks = append(blocks, block)

		if number == 0 {
			break
		}
		number--
	}

	return blocks, nil
}

func (s *Store) UpdateCanonicalHead(hash string) (ApplyResult, error) {
	if s == nil || !s.hasHead {
		return ApplyResult{}, ErrCanonicalHeadUnset
	}
	if hash == "" {
		return ApplyResult{}, ErrEmptyBlockHash
	}

	block, ok := s.blocks[hash]
	if !ok {
		return ApplyResult{}, fmt.Errorf("%w: hash=%s", ErrUnknownBlock, hash)
	}

	changed := s.canonicalHead.Hash != block.Hash
	s.canonicalHead = block
	s.rebuildCanonicalIndex(block)

	return ApplyResult{
		Accepted:         true,
		CanonicalChanged: changed,
		CanonicalHead:    s.canonicalHead,
		SafeHead:         s.safeHead,
		FinalizedHead:    s.finalizedHead,
	}, nil
}

func (s *Store) UpdateSafety(safeHash, finalizedHash string) (ApplyResult, error) {
	if s == nil || !s.hasHead {
		return ApplyResult{}, ErrCanonicalHeadUnset
	}

	if safeHash != "" {
		block, ok := s.blocks[safeHash]
		if !ok {
			return ApplyResult{}, fmt.Errorf("%w: safe=%s", ErrUnknownBlock, safeHash)
		}
		s.safeHead = block
	}

	if finalizedHash != "" {
		block, ok := s.blocks[finalizedHash]
		if !ok {
			return ApplyResult{}, fmt.Errorf("%w: finalized=%s", ErrUnknownBlock, finalizedHash)
		}
		s.finalizedHead = block
	}

	return ApplyResult{
		Accepted:         true,
		CanonicalChanged: false,
		CanonicalHead:    s.canonicalHead,
		SafeHead:         s.safeHead,
		FinalizedHead:    s.finalizedHead,
	}, nil
}

func (s *Store) Apply(block BlockRef) (ApplyResult, error) {
	if s == nil || !s.hasHead {
		return ApplyResult{}, ErrCanonicalHeadUnset
	}
	if block.Hash == "" {
		return ApplyResult{}, ErrEmptyBlockHash
	}
	if block.Number > 0 && block.ParentHash == "" {
		return ApplyResult{}, ErrEmptyParentHash
	}

	if existing, ok := s.blocks[block.Hash]; ok {
		return ApplyResult{
			Accepted:         true,
			CanonicalChanged: false,
			CanonicalHead:    s.canonicalHead,
			SafeHead:         s.safeHead,
			FinalizedHead:    s.finalizedHead,
		}, s.validateExistingBlock(existing, block)
	}

	if block.Number > 0 {
		parent, ok := s.blocks[block.ParentHash]
		if !ok {
			return ApplyResult{}, fmt.Errorf("%w: parent=%s block=%s", ErrUnknownParent, block.ParentHash, block.Hash)
		}
		if block.Number != parent.Number+1 {
			return ApplyResult{}, fmt.Errorf("%w: parent_number=%d block_number=%d", ErrBlockNumberGap, parent.Number, block.Number)
		}
	}

	s.blocks[block.Hash] = block

	changed := false
	if block.Number > s.canonicalHead.Number {
		s.canonicalHead = block
		s.rebuildCanonicalIndex(block)
		changed = true
	}

	return ApplyResult{
		Accepted:         true,
		CanonicalChanged: changed,
		CanonicalHead:    s.canonicalHead,
		SafeHead:         s.safeHead,
		FinalizedHead:    s.finalizedHead,
	}, nil
}

func (s *Store) rebuildCanonicalIndex(head BlockRef) {
	index := make(map[uint64]string)

	current := head
	for {
		index[current.Number] = current.Hash
		if current.Number == 0 || current.ParentHash == "" {
			break
		}

		parent, ok := s.blocks[current.ParentHash]
		if !ok {
			break
		}
		current = parent
	}

	s.canonicalByNum = index
}

func (s *Store) validateExistingBlock(existing, incoming BlockRef) error {
	if existing.Number != incoming.Number {
		return fmt.Errorf("%w: existing_number=%d incoming_number=%d", ErrBlockNumberGap, existing.Number, incoming.Number)
	}
	if existing.ParentHash != incoming.ParentHash {
		return fmt.Errorf("%w: existing_parent=%s incoming_parent=%s", ErrUnknownParent, existing.ParentHash, incoming.ParentHash)
	}
	return nil
}
