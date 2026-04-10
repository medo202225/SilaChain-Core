package engineapi

import (
	"errors"
	"fmt"

	"silachain/internal/consensus/forkchoice"
	"silachain/internal/consensus/txpool"
)

const (
	PayloadStatusValid   = "VALID"
	PayloadStatusInvalid = "INVALID"
	PayloadStatusSyncing = "SYNCING"
)

var (
	ErrNilStore         = errors.New("engineapi: nil forkchoice store")
	ErrEmptyPayloadHash = errors.New("engineapi: empty payload block hash")
	ErrEmptyHeadHash    = errors.New("engineapi: empty forkchoice head hash")
)

type ForkchoiceStore interface {
	HasBlock(hash string) bool
	GetBlock(hash string) (forkchoice.BlockRef, bool)
	CanonicalHead() (forkchoice.BlockRef, error)
	SafeHead() (forkchoice.BlockRef, error)
	FinalizedHead() (forkchoice.BlockRef, error)
	UpdateCanonicalHead(hash string) (forkchoice.ApplyResult, error)
	UpdateSafety(safeHash, finalizedHash string) (forkchoice.ApplyResult, error)
	Apply(block forkchoice.BlockRef) (forkchoice.ApplyResult, error)
}

type PayloadEnvelope struct {
	BlockNumber     uint64 `json:"blockNumber"`
	BlockHash       string `json:"blockHash"`
	ParentHash      string `json:"parentHash"`
	ParentStateRoot string `json:"parentStateRoot"`
	StateRoot       string `json:"stateRoot"`
}

type PayloadStatus struct {
	Status          string `json:"status"`
	LatestValidHash string `json:"latestValidHash"`
	ValidationError string `json:"validationError"`
}

type ForkchoiceState struct {
	HeadBlockHash      string `json:"headBlockHash"`
	SafeBlockHash      string `json:"safeBlockHash"`
	FinalizedBlockHash string `json:"finalizedBlockHash"`
}

type ForkchoiceUpdatedResult struct {
	PayloadStatus PayloadStatus       `json:"payloadStatus"`
	CanonicalHead forkchoice.BlockRef `json:"canonicalHead"`
	SafeHead      forkchoice.BlockRef `json:"safeHead"`
	FinalizedHead forkchoice.BlockRef `json:"finalizedHead"`
}

type Service struct {
	store    ForkchoiceStore
	buffered map[string]forkchoice.BlockRef
}

func New(store ForkchoiceStore) (*Service, error) {
	if store == nil {
		return nil, ErrNilStore
	}

	return &Service{
		store:    store,
		buffered: make(map[string]forkchoice.BlockRef),
	}, nil
}

func (s *Service) NewPayload(payload PayloadEnvelope) (PayloadStatus, error) {
	if s == nil || s.store == nil {
		return PayloadStatus{}, ErrNilStore
	}
	if payload.BlockHash == "" {
		return PayloadStatus{
			Status:          PayloadStatusInvalid,
			LatestValidHash: "",
			ValidationError: ErrEmptyPayloadHash.Error(),
		}, ErrEmptyPayloadHash
	}
	if payload.BlockNumber > 0 && payload.ParentHash == "" {
		return PayloadStatus{
			Status:          PayloadStatusInvalid,
			LatestValidHash: "",
			ValidationError: "engineapi: empty payload parent hash",
		}, nil
	}

	if s.store.HasBlock(payload.BlockHash) {
		return PayloadStatus{
			Status:          PayloadStatusValid,
			LatestValidHash: payload.BlockHash,
			ValidationError: "",
		}, nil
	}
	if _, ok := s.buffered[payload.BlockHash]; ok {
		return PayloadStatus{
			Status:          PayloadStatusValid,
			LatestValidHash: payload.BlockHash,
			ValidationError: "",
		}, nil
	}

	parent, ok := s.lookupBlock(payload.ParentHash)
	if !ok {
		return PayloadStatus{
			Status:          PayloadStatusSyncing,
			LatestValidHash: "",
			ValidationError: "",
		}, nil
	}

	if payload.BlockNumber != parent.Number+1 {
		return PayloadStatus{
			Status:          PayloadStatusInvalid,
			LatestValidHash: parent.Hash,
			ValidationError: fmt.Sprintf("engineapi: invalid block number progression parent=%d block=%d", parent.Number, payload.BlockNumber),
		}, nil
	}

	s.buffered[payload.BlockHash] = forkchoice.BlockRef{
		Number:     payload.BlockNumber,
		Hash:       payload.BlockHash,
		ParentHash: payload.ParentHash,
		StateRoot:  payload.StateRoot,
	}

	return PayloadStatus{
		Status:          PayloadStatusValid,
		LatestValidHash: payload.BlockHash,
		ValidationError: "",
	}, nil
}

func (s *Service) ForkchoiceUpdated(state ForkchoiceState) (ForkchoiceUpdatedResult, error) {
	if s == nil || s.store == nil {
		return ForkchoiceUpdatedResult{}, ErrNilStore
	}
	if state.HeadBlockHash == "" {
		return ForkchoiceUpdatedResult{
			PayloadStatus: PayloadStatus{
				Status:          PayloadStatusInvalid,
				LatestValidHash: "",
				ValidationError: ErrEmptyHeadHash.Error(),
			},
		}, ErrEmptyHeadHash
	}

	if !s.store.HasBlock(state.HeadBlockHash) {
		if _, ok := s.buffered[state.HeadBlockHash]; !ok {
			head, err := s.store.CanonicalHead()
			if err != nil {
				return ForkchoiceUpdatedResult{}, err
			}
			safe, err := s.store.SafeHead()
			if err != nil {
				return ForkchoiceUpdatedResult{}, err
			}
			finalized, err := s.store.FinalizedHead()
			if err != nil {
				return ForkchoiceUpdatedResult{}, err
			}

			return ForkchoiceUpdatedResult{
				PayloadStatus: PayloadStatus{
					Status:          PayloadStatusSyncing,
					LatestValidHash: head.Hash,
					ValidationError: "",
				},
				CanonicalHead: head,
				SafeHead:      safe,
				FinalizedHead: finalized,
			}, nil
		}

		if err := s.commitBufferedChain(state.HeadBlockHash); err != nil {
			head, headErr := s.store.CanonicalHead()
			if headErr != nil {
				return ForkchoiceUpdatedResult{}, err
			}
			safe, _ := s.store.SafeHead()
			finalized, _ := s.store.FinalizedHead()

			return ForkchoiceUpdatedResult{
				PayloadStatus: PayloadStatus{
					Status:          PayloadStatusInvalid,
					LatestValidHash: head.Hash,
					ValidationError: err.Error(),
				},
				CanonicalHead: head,
				SafeHead:      safe,
				FinalizedHead: finalized,
			}, nil
		}
	}

	applyResult, err := s.store.UpdateCanonicalHead(state.HeadBlockHash)
	if err != nil {
		return ForkchoiceUpdatedResult{}, err
	}

	safetyResult, err := s.store.UpdateSafety(state.SafeBlockHash, state.FinalizedBlockHash)
	if err != nil {
		return ForkchoiceUpdatedResult{}, err
	}

	return ForkchoiceUpdatedResult{
		PayloadStatus: PayloadStatus{
			Status:          PayloadStatusValid,
			LatestValidHash: applyResult.CanonicalHead.Hash,
			ValidationError: "",
		},
		CanonicalHead: applyResult.CanonicalHead,
		SafeHead:      safetyResult.SafeHead,
		FinalizedHead: safetyResult.FinalizedHead,
	}, nil
}

func (s *Service) lookupBlock(hash string) (forkchoice.BlockRef, bool) {
	if hash == "" {
		return forkchoice.BlockRef{}, false
	}

	if block, ok := s.store.GetBlock(hash); ok {
		return block, true
	}
	block, ok := s.buffered[hash]
	return block, ok
}

func (s *Service) commitBufferedChain(headHash string) error {
	ordered, err := s.collectMissingChain(headHash)
	if err != nil {
		return err
	}

	for _, block := range ordered {
		if _, err := s.store.Apply(block); err != nil {
			return err
		}
		delete(s.buffered, block.Hash)
	}

	return nil
}

func (s *Service) collectMissingChain(headHash string) ([]forkchoice.BlockRef, error) {
	currentHash := headHash
	collected := make([]forkchoice.BlockRef, 0)

	for {
		if s.store.HasBlock(currentHash) {
			break
		}

		block, ok := s.buffered[currentHash]
		if !ok {
			return nil, fmt.Errorf("engineapi: missing buffered block %s", currentHash)
		}

		collected = append(collected, block)
		currentHash = block.ParentHash
	}

	for left, right := 0, len(collected)-1; left < right; left, right = left+1, right-1 {
		collected[left], collected[right] = collected[right], collected[left]
	}

	return collected, nil
}

func CanonicalTransactionsForPayload(meta PayloadMetadata) []txpool.Tx {
	return meta.Transactions
}
