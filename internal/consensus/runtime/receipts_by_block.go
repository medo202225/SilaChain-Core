package runtime

import "errors"

var (
	ErrNilRuntimeReceiptsByBlock = errors.New("runtime: nil runtime receipts by block")
	ErrNilAPIReceiptsByBlock     = errors.New("runtime: nil api receipts by block")
)

type ChainReceiptsByBlockResult struct {
	Receipts    []ChainReceiptResult `json:"receipts"`
	BlockHash   string               `json:"blockHash"`
	BlockNumber uint64               `json:"blockNumber"`
	Found       bool                 `json:"found"`
}

func (r *Runtime) ChainReceiptsByBlock(hash string) (ChainReceiptsByBlockResult, error) {
	if r == nil {
		return ChainReceiptsByBlockResult{}, ErrNilRuntimeReceiptsByBlock
	}
	if r.api == nil {
		return ChainReceiptsByBlockResult{}, ErrNilAPIReceiptsByBlock
	}

	receipts, ok := r.api.CanonicalReceiptsByBlockHash(hash)
	if !ok {
		return ChainReceiptsByBlockResult{
			Found: false,
		}, nil
	}

	blockResult, err := r.ChainBlock(hash)
	if err != nil {
		return ChainReceiptsByBlockResult{}, err
	}
	if !blockResult.Found {
		return ChainReceiptsByBlockResult{
			Found: false,
		}, nil
	}

	out := make([]ChainReceiptResult, 0, len(receipts))
	for _, receipt := range receipts {
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

		out = append(out, ChainReceiptResult{
			TxHash:      receipt.TxHash,
			BlockHash:   receipt.BlockHash,
			BlockNumber: receipt.BlockNumber,
			GasUsed:     receipt.GasUsed,
			Success:     receipt.Success,
			Logs:        logs,
			Found:       true,
		})
	}

	return ChainReceiptsByBlockResult{
		Receipts:    out,
		BlockHash:   blockResult.Block.Hash,
		BlockNumber: blockResult.Block.Number,
		Found:       true,
	}, nil
}

func (r *Runtime) ChainReceiptsByBlockNumber(number uint64) (ChainReceiptsByBlockResult, error) {
	if r == nil {
		return ChainReceiptsByBlockResult{}, ErrNilRuntimeReceiptsByBlock
	}
	if r.api == nil {
		return ChainReceiptsByBlockResult{}, ErrNilAPIReceiptsByBlock
	}

	receipts, ok := r.api.CanonicalReceiptsByBlockNumber(number)
	if !ok {
		return ChainReceiptsByBlockResult{
			Found: false,
		}, nil
	}

	blockResult, err := r.ChainBlockByNumber(number)
	if err != nil {
		return ChainReceiptsByBlockResult{}, err
	}
	if !blockResult.Found {
		return ChainReceiptsByBlockResult{
			Found: false,
		}, nil
	}

	out := make([]ChainReceiptResult, 0, len(receipts))
	for _, receipt := range receipts {
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

		out = append(out, ChainReceiptResult{
			TxHash:      receipt.TxHash,
			BlockHash:   receipt.BlockHash,
			BlockNumber: receipt.BlockNumber,
			GasUsed:     receipt.GasUsed,
			Success:     receipt.Success,
			Logs:        logs,
			Found:       true,
		})
	}

	return ChainReceiptsByBlockResult{
		Receipts:    out,
		BlockHash:   blockResult.Block.Hash,
		BlockNumber: blockResult.Block.Number,
		Found:       true,
	}, nil
}
