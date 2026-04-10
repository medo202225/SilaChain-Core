package block

import (
	"time"

	coretypes "silachain/internal/core/types"
	pkgtypes "silachain/pkg/types"
)

func NewBlock(
	height pkgtypes.Height,
	parentHash pkgtypes.Hash,
	stateRoot pkgtypes.Hash,
	txRoot pkgtypes.Hash,
	receiptRoot pkgtypes.Hash,
	proposer pkgtypes.Address,
	gasUsed pkgtypes.Gas,
	gasLimit pkgtypes.Gas,
	transactions []coretypes.Transaction,
	receipts []coretypes.Receipt,
) (*coretypes.Block, error) {
	computedTxRoot, err := TxRootHash(transactions)
	if err != nil {
		return nil, err
	}

	computedReceiptRoot, err := ReceiptRootHash(receipts)
	if err != nil {
		return nil, err
	}

	h := coretypes.Header{
		Height:      height,
		ParentHash:  parentHash,
		StateRoot:   stateRoot,
		TxRoot:      computedTxRoot,
		ReceiptRoot: computedReceiptRoot,
		Timestamp:   pkgtypes.Timestamp(time.Now().Unix()),
		Proposer:    proposer,
		GasUsed:     gasUsed,
		GasLimit:    gasLimit,
		TxCount:     uint64(len(transactions)),
	}

	hash, err := HeaderHash(h)
	if err != nil {
		return nil, err
	}
	h.Hash = hash

	return &coretypes.Block{
		Header:       h,
		Transactions: transactions,
		Receipts:     receipts,
	}, nil
}
