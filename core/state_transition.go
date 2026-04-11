package core

import (
	"fmt"
	"silachain/internal/consensus/txpool"

	"silachain/internal/execution/executionstate"
)

var ErrNilStateTransition = fmt.Errorf("core: nil state transition")

type StateTransition struct {
	state *executionstate.State
}

func NewStateTransition(state *executionstate.State) *StateTransition {
	return &StateTransition{
		state: state,
	}
}

func (st *StateTransition) ApplyTransaction(tx executionstate.PendingTx) (executionstate.Receipt, error) {
	if st == nil || st.state == nil {
		return executionstate.Receipt{}, ErrNilStateTransition
	}

	return st.state.ApplyTransactionInBlock(tx, 0, "")
}
func PendingTxFromPoolTx(tx txpool.Tx, baseFee uint64) executionstate.PendingTx {
	return executionstate.PendingTx{
		Hash:     tx.Hash,
		From:     tx.From,
		To:       "SILA_BLOCK_FEE_SINK",
		Value:    0,
		Nonce:    tx.Nonce,
		Data:     "",
		Fee:      tx.EffectiveFee(baseFee),
		GasLimit: tx.GasLimit,
	}
}

func PendingTxsFromPoolTxs(txs []txpool.Tx, baseFee uint64) []executionstate.PendingTx {
	out := make([]executionstate.PendingTx, 0, len(txs))
	for _, tx := range txs {
		out = append(out, PendingTxFromPoolTx(tx, baseFee))
	}
	return out
}
func (st *StateTransition) ApplyTransactions(txs []executionstate.PendingTx) ([]executionstate.Receipt, uint64, error) {
	receipts := make([]executionstate.Receipt, 0, len(txs))
	var gasUsed uint64

	for _, tx := range txs {
		receipt, err := st.ApplyTransaction(tx)
		if err != nil {
			return nil, 0, err
		}
		gasUsed += receipt.GasUsed
		receipts = append(receipts, receipt)
	}

	return receipts, gasUsed, nil
}

func ReceiptsFromExecutionResult(executed executionstate.BlockExecutionResult, txs []txpool.Tx) []Receipt {
	receipts := make([]Receipt, 0, len(executed.Receipts))
	for _, receipt := range executed.Receipts {
		receipts = append(receipts, Receipt{
			TxHash:  receipt.TxHash,
			From:    receipt.From,
			Nonce:   findTxNonce(txs, receipt.TxHash),
			GasUsed: receipt.GasUsed,
			Success: receipt.Success,
		})
	}
	return receipts
}

func (st *StateTransition) ApplyBlockTransactions(block executionstate.ImportedBlock, txs []executionstate.PendingTx) ([]executionstate.Receipt, uint64, error) {
	if st == nil || st.state == nil {
		return nil, 0, ErrNilStateTransition
	}
	if len(block.TxHashes) != len(txs) {
		return nil, 0, fmt.Errorf("execution state: tx count mismatch for block")
	}

	receipts := make([]executionstate.Receipt, 0, len(txs))
	var totalGasUsed uint64

	for i, tx := range txs {
		if block.TxHashes[i] != tx.Hash {
			return nil, 0, fmt.Errorf("execution state: tx hash mismatch at index %d", i)
		}

		tx = executionstate.NormalizeTx(tx)
		gasUsed := executionstate.IntrinsicGas(tx)
		if totalGasUsed+gasUsed > executionstate.DefaultBlockGasLimit {
			return nil, 0, fmt.Errorf("execution state: block gas limit exceeded")
		}

		receipt, err := st.state.ApplyTransactionInBlock(tx, block.Number, block.Hash)
		if err != nil {
			return nil, 0, err
		}

		totalGasUsed += receipt.GasUsed
		receipts = append(receipts, receipt)
	}

	return receipts, totalGasUsed, nil
}
