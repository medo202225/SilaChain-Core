package block

import (
	"errors"

	coretypes "silachain/internal/core/types"
	pkgtypes "silachain/pkg/types"
)

var (
	ErrNilBlock           = errors.New("block is nil")
	ErrInvalidBlockHash   = errors.New("invalid block hash")
	ErrInvalidTxCount     = errors.New("invalid transaction count")
	ErrInvalidBlockBody   = errors.New("invalid block body")
	ErrDuplicateBlockTx   = errors.New("duplicate transaction in block")
	ErrInvalidTxRoot      = errors.New("invalid tx root")
	ErrInvalidReceiptRoot = errors.New("invalid receipt root")
)

func Validate(b *coretypes.Block) error {
	if b == nil {
		return ErrNilBlock
	}

	if b.Header.TxCount != uint64(len(b.Transactions)) {
		return ErrInvalidTxCount
	}

	seen := make(map[pkgtypes.Hash]struct{}, len(b.Transactions))
	for _, txn := range b.Transactions {
		if txn.Hash == "" {
			return ErrInvalidBlockBody
		}
		if _, ok := seen[txn.Hash]; ok {
			return ErrDuplicateBlockTx
		}
		seen[txn.Hash] = struct{}{}
	}

	expectedTxRoot, err := TxRootHash(b.Transactions)
	if err != nil {
		return err
	}
	if b.Header.TxRoot != expectedTxRoot {
		return ErrInvalidTxRoot
	}

	expectedReceiptRoot, err := ReceiptRootHash(b.Receipts)
	if err != nil {
		return err
	}
	if b.Header.ReceiptRoot != expectedReceiptRoot {
		return ErrInvalidReceiptRoot
	}

	expectedHash, err := HeaderHash(b.Header)
	if err != nil {
		return err
	}

	if b.Header.Hash != expectedHash {
		return ErrInvalidBlockHash
	}

	return nil
}
