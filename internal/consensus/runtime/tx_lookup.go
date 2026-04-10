package runtime

import (
	"errors"

	"silachain/internal/consensus/txpool"
)

var (
	ErrNilRuntimeTransactionLookup = errors.New("runtime: nil runtime transaction lookup")
	ErrNilAPITransactionLookup     = errors.New("runtime: nil api for transaction lookup")
)

type ChainTransactionResult struct {
	Transaction txpool.Tx `json:"transaction"`
	BlockHash   string    `json:"blockHash"`
	BlockNumber uint64    `json:"blockNumber"`
	Found       bool      `json:"found"`
}

func (r *Runtime) ChainTransaction(hash string) (ChainTransactionResult, error) {
	if r == nil {
		return ChainTransactionResult{}, ErrNilRuntimeTransactionLookup
	}
	if r.api == nil {
		return ChainTransactionResult{}, ErrNilAPITransactionLookup
	}

	meta, tx, ok := r.api.CanonicalTransactionByHash(hash)
	if !ok {
		return ChainTransactionResult{
			Found: false,
		}, nil
	}

	return ChainTransactionResult{
		Transaction: tx,
		BlockHash:   meta.BlockHash,
		BlockNumber: meta.BlockNumber,
		Found:       true,
	}, nil
}
