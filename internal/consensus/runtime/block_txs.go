package runtime

import (
	"errors"

	"silachain/internal/consensus/txpool"
)

var (
	ErrNilRuntimeBlockTxs = errors.New("runtime: nil runtime block txs")
	ErrNilAPIBlockTxs     = errors.New("runtime: nil api block txs")
)

type ChainBlockTransactionsResult struct {
	Transactions []txpool.Tx `json:"transactions"`
	BlockHash    string      `json:"blockHash"`
	BlockNumber  uint64      `json:"blockNumber"`
	Found        bool        `json:"found"`
}

func (r *Runtime) ChainBlockTransactions(hash string) (ChainBlockTransactionsResult, error) {
	if r == nil {
		return ChainBlockTransactionsResult{}, ErrNilRuntimeBlockTxs
	}
	if r.api == nil {
		return ChainBlockTransactionsResult{}, ErrNilAPIBlockTxs
	}

	txs, ok := r.api.GetCanonicalBlockTransactionsByHash(hash)
	if !ok {
		return ChainBlockTransactionsResult{
			Found: false,
		}, nil
	}

	blockResult, err := r.ChainBlock(hash)
	if err != nil {
		return ChainBlockTransactionsResult{}, err
	}
	if !blockResult.Found {
		return ChainBlockTransactionsResult{
			Found: false,
		}, nil
	}

	return ChainBlockTransactionsResult{
		Transactions: txs,
		BlockHash:    blockResult.Block.Hash,
		BlockNumber:  blockResult.Block.Number,
		Found:        true,
	}, nil
}

func (r *Runtime) ChainBlockTransactionsByNumber(number uint64) (ChainBlockTransactionsResult, error) {
	if r == nil {
		return ChainBlockTransactionsResult{}, ErrNilRuntimeBlockTxs
	}
	if r.api == nil {
		return ChainBlockTransactionsResult{}, ErrNilAPIBlockTxs
	}

	txs, ok := r.api.GetCanonicalBlockTransactionsByNumber(number)
	if !ok {
		return ChainBlockTransactionsResult{
			Found: false,
		}, nil
	}

	blockResult, err := r.ChainBlockByNumber(number)
	if err != nil {
		return ChainBlockTransactionsResult{}, err
	}
	if !blockResult.Found {
		return ChainBlockTransactionsResult{
			Found: false,
		}, nil
	}

	return ChainBlockTransactionsResult{
		Transactions: txs,
		BlockHash:    blockResult.Block.Hash,
		BlockNumber:  blockResult.Block.Number,
		Found:        true,
	}, nil
}
