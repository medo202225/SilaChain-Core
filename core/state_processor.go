package core

import (
	"errors"
	"fmt"
	"strings"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/txpool"
	"silachain/internal/execution/executionstate"
)

var (
	ErrNilState        = errors.New("payloadexecution: nil execution state")
	ErrNilAssembler    = errors.New("payloadexecution: nil assembler")
	ErrEmptyParentHash = errors.New("payloadexecution: empty parent hash")
	ErrInvalidBlockNum = errors.New("payloadexecution: invalid block number")
)

type State interface {
	Head() blockassembly.Head
	ExecuteBlock(req executionstate.BlockExecutionRequest) (executionstate.BlockExecutionResult, error)
	ExecutionState() *executionstate.State
}

type StateProcessor struct {
	state     State
	assembler *blockassembly.Assembler
}

type Receipt struct {
	TxHash  string
	From    string
	Nonce   uint64
	GasUsed uint64
	Success bool
}

type Result struct {
	BlockNumber        uint64
	BlockHash          string
	ParentHash         string
	ExecutionStateRoot string
	BaseFee            uint64
	GasUsed            uint64
	Receipts           []Receipt
	TxCount            int
}

func NewStateProcessor(state State, assembler *blockassembly.Assembler) (*StateProcessor, error) {
	if state == nil {
		return nil, ErrNilState
	}
	if assembler == nil {
		return nil, ErrNilAssembler
	}
	return &StateProcessor{
		state:     state,
		assembler: assembler,
	}, nil
}

func (p *StateProcessor) Process(attrs blockassembly.PayloadAttributes) (Result, error) {
	if p == nil || p.state == nil {
		return Result{}, ErrNilState
	}
	if p.assembler == nil {
		return Result{}, ErrNilAssembler
	}

	assembled, err := p.assembler.Assemble(attrs)
	if err != nil {
		return Result{}, err
	}
	if assembled.ParentHash == "" {
		return Result{}, ErrEmptyParentHash
	}
	if assembled.BlockNumber != assembled.ParentNumber+1 {
		return Result{}, fmt.Errorf("%w: parent=%d block=%d", ErrInvalidBlockNum, assembled.ParentNumber, assembled.BlockNumber)
	}

	blockHash := deriveBlockHash(assembled)

	execTxs := PendingTxsFromPoolTxs(assembled.Selection.Transactions, assembled.BaseFee)

	executed, err := p.state.ExecuteBlock(executionstate.BlockExecutionRequest{
		Block: executionstate.ImportedBlock{
			Number:     assembled.BlockNumber,
			Hash:       blockHash,
			ParentHash: assembled.ParentHash,
			Timestamp:  attrs.Timestamp,
			TxHashes:   collectTxHashes(assembled.Selection.Transactions),
		},
		Txs: execTxs,
	})
	if err != nil {
		return Result{}, err
	}

	receipts := ReceiptsFromExecutionResult(executed, assembled.Selection.Transactions)

	return Result{
		BlockNumber:        assembled.BlockNumber,
		BlockHash:          blockHash,
		ParentHash:         assembled.ParentHash,
		ExecutionStateRoot: executed.StateRoot,
		BaseFee:            assembled.BaseFee,
		GasUsed:            executed.GasUsed,
		Receipts:           receipts,
		TxCount:            len(receipts),
	}, nil
}

func deriveBlockHash(assembled blockassembly.Result) string {
	return fmt.Sprintf(
		"sila-block-%d-%s-%d",
		assembled.BlockNumber,
		sanitizeHashComponent(assembled.ParentHash),
		len(assembled.Selection.Transactions),
	)
}

func sanitizeHashComponent(v string) string {
	replacer := strings.NewReplacer(":", "-", "/", "-", "\\", "-", " ", "-")
	return replacer.Replace(v)
}

func TxToPoolTx(hash, from string, nonce, gasLimit, maxFeePerGas, maxPriorityFeePerGas uint64, timestamp int64) txpool.Tx {
	return txpool.Tx{
		Hash:                 hash,
		From:                 from,
		Nonce:                nonce,
		GasLimit:             gasLimit,
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		Timestamp:            timestamp,
	}
}

func collectTxHashes(txs []txpool.Tx) []string {
	out := make([]string, 0, len(txs))
	for _, tx := range txs {
		out = append(out, tx.Hash)
	}
	return out
}

func findTxNonce(txs []txpool.Tx, hash string) uint64 {
	for _, tx := range txs {
		if tx.Hash == hash {
			return tx.Nonce
		}
	}
	return 0
}
