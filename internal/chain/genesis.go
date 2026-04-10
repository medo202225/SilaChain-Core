package chain

import (
	block "silachain/internal/block"
	coretypes "silachain/internal/core/types"
	pkgtypes "silachain/pkg/types"
)

func NewGenesisBlock() (*coretypes.Block, error) {
	b, err := block.NewBlock(
		0,
		"",
		"",
		"",
		"",
		"",
		0,
		0,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	b.Header.Timestamp = 0

	txRoot, err := block.TxRootHash(b.Transactions)
	if err != nil {
		return nil, err
	}
	b.Header.TxRoot = txRoot

	receiptRoot, err := block.ReceiptRootHash(b.Receipts)
	if err != nil {
		return nil, err
	}
	b.Header.ReceiptRoot = receiptRoot

	hash, err := block.HeaderHash(b.Header)
	if err != nil {
		return nil, err
	}
	b.Header.Hash = hash

	return b, nil
}

func IsGenesisBlock(b *coretypes.Block) bool {
	if b == nil {
		return false
	}
	return b.Header.Height == pkgtypes.Height(0)
}
