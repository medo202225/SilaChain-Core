package runtime

import (
	"errors"
)

var (
	ErrNilRuntimeReceiptLookup = errors.New("runtime: nil runtime receipt lookup")
	ErrNilAPIReceiptLookup     = errors.New("runtime: nil api receipt lookup")
)

type ChainReceiptLog struct {
	TxHash      string   `json:"txHash"`
	BlockHash   string   `json:"blockHash"`
	BlockNumber uint64   `json:"blockNumber"`
	LogIndex    uint64   `json:"logIndex"`
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
}

type ChainReceiptResult struct {
	TxHash      string            `json:"txHash"`
	BlockHash   string            `json:"blockHash"`
	BlockNumber uint64            `json:"blockNumber"`
	GasUsed     uint64            `json:"gasUsed"`
	Success     bool              `json:"success"`
	Logs        []ChainReceiptLog `json:"logs"`
	Found       bool              `json:"found"`
}

func mapReceiptLogs(receiptLogs []struct {
	TxHash      string
	BlockHash   string
	BlockNumber uint64
	LogIndex    uint64
	Address     string
	Topics      []string
	Data        string
}) []ChainReceiptLog {
	out := make([]ChainReceiptLog, 0, len(receiptLogs))
	for _, logEntry := range receiptLogs {
		topics := make([]string, len(logEntry.Topics))
		copy(topics, logEntry.Topics)

		out = append(out, ChainReceiptLog{
			TxHash:      logEntry.TxHash,
			BlockHash:   logEntry.BlockHash,
			BlockNumber: logEntry.BlockNumber,
			LogIndex:    logEntry.LogIndex,
			Address:     logEntry.Address,
			Topics:      topics,
			Data:        logEntry.Data,
		})
	}
	return out
}

// ChainReceipt currently resolves through canonical engineapi metadata lookups.
// Restart-stable receipt/tx/log lookup should be re-bridged to persisted chain readers
// because chain storage already persists tx index, receipts, and log-query inputs across reload.

func (r *Runtime) ChainReceipt(txHash string) (ChainReceiptResult, error) {
	if r == nil {
		return ChainReceiptResult{}, ErrNilRuntimeReceiptLookup
	}

	if r.receiptReader != nil {
		persisted, ok := r.receiptReader.GetReceiptByHash(txHash)
		if ok && persisted != nil {
			logs := make([]ChainReceiptLog, 0, len(persisted.Logs))
			for i, logEntry := range persisted.Logs {
				topics := append([]string(nil), logEntry.Topics...)
				logs = append(logs, ChainReceiptLog{
					TxHash:      string(persisted.TxHash),
					BlockHash:   string(persisted.BlockHash),
					BlockNumber: uint64(persisted.BlockHeight),
					LogIndex:    uint64(i),
					Address:     logEntry.Address,
					Topics:      topics,
					Data:        "",
				})
			}

			return ChainReceiptResult{
				TxHash:      string(persisted.TxHash),
				BlockHash:   string(persisted.BlockHash),
				BlockNumber: uint64(persisted.BlockHeight),
				GasUsed:     uint64(persisted.GasUsed),
				Success:     persisted.Success,
				Logs:        logs,
				Found:       true,
			}, nil
		}
	}

	if r.api == nil {
		return ChainReceiptResult{}, ErrNilAPIReceiptLookup
	}

	receipt, ok := r.api.CanonicalReceiptByTxHash(txHash)
	if !ok {
		return ChainReceiptResult{
			Found: false,
		}, nil
	}

	logs := make([]ChainReceiptLog, 0, len(receipt.Logs))
	for _, logEntry := range receipt.Logs {
		topics := make([]string, len(logEntry.Topics))
		copy(topics, logEntry.Topics)

		logs = append(logs, ChainReceiptLog{
			TxHash:      logEntry.TxHash,
			BlockHash:   logEntry.BlockHash,
			BlockNumber: logEntry.BlockNumber,
			LogIndex:    logEntry.LogIndex,
			Address:     logEntry.Address,
			Topics:      topics,
			Data:        logEntry.Data,
		})
	}

	return ChainReceiptResult{
		TxHash:      receipt.TxHash,
		BlockHash:   receipt.BlockHash,
		BlockNumber: receipt.BlockNumber,
		GasUsed:     receipt.GasUsed,
		Success:     receipt.Success,
		Logs:        logs,
		Found:       true,
	}, nil
}
