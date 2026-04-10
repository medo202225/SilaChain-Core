package runtime

import "errors"

var (
	ErrNilRuntimeLogsLookup = errors.New("runtime: nil runtime logs lookup")
	ErrNilAPILogsLookup     = errors.New("runtime: nil api logs lookup")
)

type ChainLogsResult struct {
	Logs        []any  `json:"logs"`
	BlockHash   string `json:"blockHash,omitempty"`
	BlockNumber uint64 `json:"blockNumber,omitempty"`
	TxHash      string `json:"txHash,omitempty"`
	Found       bool   `json:"found"`
}

func (r *Runtime) ChainLogs(txHash string) (ChainLogsResult, error) {
	if r == nil {
		return ChainLogsResult{}, ErrNilRuntimeLogsLookup
	}
	if r.api == nil {
		return ChainLogsResult{}, ErrNilAPILogsLookup
	}

	logs, ok := r.api.CanonicalLogsByTxHash(txHash)
	if !ok {
		return ChainLogsResult{Found: false}, nil
	}

	receipt, err := r.ChainReceipt(txHash)
	if err != nil {
		return ChainLogsResult{}, err
	}
	if !receipt.Found {
		return ChainLogsResult{Found: false}, nil
	}

	out := make([]any, 0, len(logs))
	for _, log := range logs {
		out = append(out, log)
	}

	return ChainLogsResult{
		Logs:        out,
		BlockHash:   receipt.BlockHash,
		BlockNumber: receipt.BlockNumber,
		TxHash:      txHash,
		Found:       true,
	}, nil
}

func (r *Runtime) ChainLogsByBlock(hash string) (ChainLogsResult, error) {
	if r == nil {
		return ChainLogsResult{}, ErrNilRuntimeLogsLookup
	}
	if r.api == nil {
		return ChainLogsResult{}, ErrNilAPILogsLookup
	}

	logs, ok := r.api.CanonicalLogsByBlockHash(hash)
	if !ok {
		return ChainLogsResult{Found: false}, nil
	}

	block, err := r.ChainBlock(hash)
	if err != nil {
		return ChainLogsResult{}, err
	}
	if !block.Found {
		return ChainLogsResult{Found: false}, nil
	}

	out := make([]any, 0, len(logs))
	for _, log := range logs {
		out = append(out, log)
	}

	return ChainLogsResult{
		Logs:        out,
		BlockHash:   block.Block.Hash,
		BlockNumber: block.Block.Number,
		Found:       true,
	}, nil
}

func (r *Runtime) ChainLogsByBlockNumber(number uint64) (ChainLogsResult, error) {
	if r == nil {
		return ChainLogsResult{}, ErrNilRuntimeLogsLookup
	}
	if r.api == nil {
		return ChainLogsResult{}, ErrNilAPILogsLookup
	}

	logs, ok := r.api.CanonicalLogsByBlockNumber(number)
	if !ok {
		return ChainLogsResult{Found: false}, nil
	}

	block, err := r.ChainBlockByNumber(number)
	if err != nil {
		return ChainLogsResult{}, err
	}
	if !block.Found {
		return ChainLogsResult{Found: false}, nil
	}

	out := make([]any, 0, len(logs))
	for _, log := range logs {
		out = append(out, log)
	}

	return ChainLogsResult{
		Logs:        out,
		BlockHash:   block.Block.Hash,
		BlockNumber: block.Block.Number,
		Found:       true,
	}, nil
}
