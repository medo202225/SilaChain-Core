package forkchoice

import (
	"errors"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/blockimport"
)

var (
	ErrNilImporter = errors.New("forkchoice: nil importer")
	ErrNilStore    = errors.New("forkchoice: nil store")
)

type Importer interface {
	Import(req blockimport.ImportRequest) (blockimport.Result, error)
}

type Manager struct {
	importer Importer
	store    *Store
}

type ImportAndApplyResult struct {
	Import     blockimport.Result
	ForkChoice ApplyResult
}

func NewManager(importer Importer, store *Store) (*Manager, error) {
	if importer == nil {
		return nil, ErrNilImporter
	}
	if store == nil {
		return nil, ErrNilStore
	}

	return &Manager{
		importer: importer,
		store:    store,
	}, nil
}

func (m *Manager) ImportAndApply(req blockimport.ImportRequest) (ImportAndApplyResult, error) {
	if m == nil || m.importer == nil {
		return ImportAndApplyResult{}, ErrNilImporter
	}
	if m.store == nil {
		return ImportAndApplyResult{}, ErrNilStore
	}

	importResult, err := m.importer.Import(req)
	if err != nil {
		return ImportAndApplyResult{}, err
	}

	applyResult, err := m.store.Apply(BlockRef{
		Number:     importResult.BlockNumber,
		Hash:       importResult.BlockHash,
		ParentHash: importResult.ParentHash,
		StateRoot:  importResult.StateRoot,
	})
	if err != nil {
		return ImportAndApplyResult{}, err
	}

	return ImportAndApplyResult{
		Import:     importResult,
		ForkChoice: applyResult,
	}, nil
}

func GenesisFromHead(head blockassembly.Head) BlockRef {
	return BlockRef{
		Number:     head.Number,
		Hash:       head.Hash,
		ParentHash: "",
		StateRoot:  head.StateRoot,
	}
}
