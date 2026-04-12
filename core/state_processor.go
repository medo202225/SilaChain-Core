package core

import (
	"errors"
	"fmt"
	"strings"

	statecore "silachain/core/state"
	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/txpool"
)

var (
	ErrNilState        = errors.New("payloadexecution: nil execution state")
	ErrNilAssembler    = errors.New("payloadexecution: nil assembler")
	ErrEmptyParentHash = errors.New("payloadexecution: empty parent hash")
	ErrInvalidBlockNum = errors.New("payloadexecution: invalid block number")
	ErrBlockGasLimit   = errors.New("payloadexecution: block gas limit exceeded")
)

type State interface {
	Head() blockassembly.Head
	StateDB() *statecore.StateDB
	SetHead(head blockassembly.Head) error
}

type StateProcessor struct {
	state     State
	assembler *blockassembly.Assembler
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

	db := p.state.StateDB()
	if db == nil {
		return Result{}, ErrNilStateDB
	}

	ctx := NewBlockContext(
		assembled.BlockNumber,
		deriveBlockHash(assembled),
		assembled.ParentHash,
		assembled.BaseFee,
		assembled.GasLimit,
		attrs.Timestamp,
	)

	gp := NewGasPool(ctx.GasLimit)
	receipts, totalGasUsed, successCount, failureCount, err := p.applyTransactions(ctx, db, gp, assembled.Selection.Transactions)
	if err != nil {
		return Result{}, err
	}

	stateRoot, err := db.Commit(false)
	if err != nil {
		return Result{}, err
	}

	newHead := blockassembly.Head{
		Number:    ctx.BlockNumber,
		Hash:      ctx.BlockHash,
		StateRoot: stateRoot,
		BaseFee:   ctx.BaseFee,
	}
	if err := p.state.SetHead(newHead); err != nil {
		return Result{}, err
	}

	return Result{
		BlockNumber:        ctx.BlockNumber,
		BlockHash:          ctx.BlockHash,
		ParentHash:         ctx.ParentHash,
		ExecutionStateRoot: stateRoot,
		BaseFee:            ctx.BaseFee,
		GasUsed:            totalGasUsed,
		Receipts:           receipts,
		TxCount:            len(receipts),
		SuccessCount:       successCount,
		FailureCount:       failureCount,
	}, nil
}

func (p *StateProcessor) applyTransactions(
	ctx BlockContext,
	db *statecore.StateDB,
	gp *GasPool,
	txs []txpool.Tx,
) ([]Receipt, uint64, int, int, error) {
	receipts := make([]Receipt, 0, len(txs))
	var totalGasUsed uint64
	var successCount int
	var failureCount int

	for _, tx := range txs {
		snapshot := db.Snapshot()

		msg := PoolTxToMessage(
			tx.Hash,
			tx.From,
			"SILA_BLOCK_FEE_SINK",
			tx.Nonce,
			0,
			tx.GasLimit,
			tx.EffectiveFee(ctx.BaseFee),
			nil,
		)

		receipt, err := ApplyTransaction(ctx, db, gp, tx.Hash, msg)
		if err != nil {
			db.RevertToSnapshot(snapshot)
			receipts = append(receipts, receipt)
			failureCount++
			continue
		}

		if totalGasUsed+receipt.GasUsed > ctx.GasLimit {
			db.RevertToSnapshot(snapshot)
			return nil, 0, 0, 0, ErrBlockGasLimit
		}

		totalGasUsed += receipt.GasUsed
		receipts = append(receipts, receipt)
		successCount++
	}

	return receipts, totalGasUsed, successCount, failureCount, nil
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
