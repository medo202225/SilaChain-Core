package miner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"silachain/internal/consensus/blockassembly"
)

var (
	ErrWorkerEmptyParentHash    = errors.New("miner: worker empty parent hash")
	ErrWorkerInvalidBlockNumber = errors.New("miner: worker invalid block number")
)

type newPayloadResult struct {
	err    error
	result ExecutableData
}

type generateParams struct {
	timestamp  uint64
	parentHash string
	coinbase   string
	random     string
	gasLimit   uint64
	noTxs      bool
}

func (m *Miner) generateWork(ctx context.Context, genParams *generateParams) *newPayloadResult {
	if m == nil {
		return &newPayloadResult{err: errors.New("miner: nil miner")}
	}
	if genParams == nil {
		return &newPayloadResult{err: errors.New("miner: nil generate params")}
	}
	if genParams.parentHash == "" {
		return &newPayloadResult{err: ErrWorkerEmptyParentHash}
	}

	args := BuildPayloadArgs{
		ParentHash:   genParams.parentHash,
		Timestamp:    genParams.timestamp,
		FeeRecipient: genParams.coinbase,
		Random:       genParams.random,
		GasLimit:     genParams.gasLimit,
		Version:      1,
	}

	built, stateRoot, err := m.prepareWork(genParams)
	if err != nil {
		return &newPayloadResult{err: err}
	}

	payload, err := m.BuildPayload(ctx, args, built, stateRoot)
	if err != nil {
		return &newPayloadResult{err: err}
	}

	return &newPayloadResult{
		result: payload.Resolve(),
	}
}

func (m *Miner) prepareWork(genParams *generateParams) (blockassembly.Result, string, error) {
	if genParams == nil {
		return blockassembly.Result{}, "", errors.New("miner: nil generate params")
	}
	if genParams.parentHash == "" {
		return blockassembly.Result{}, "", ErrWorkerEmptyParentHash
	}

	cfg := m.Config()
	gasLimit := genParams.gasLimit
	if gasLimit == 0 {
		gasLimit = cfg.GasCeil
	}
	if gasLimit == 0 {
		gasLimit = DefaultConfig.GasCeil
	}

	built := blockassembly.Result{
		ParentNumber:    0,
		BlockNumber:     1,
		ParentHash:      genParams.parentHash,
		ParentStateRoot: fmt.Sprintf("sila-parent-state-%s", genParams.parentHash),
		BaseFee:         1,
		GasLimit:        gasLimit,
		Attributes: blockassembly.PayloadAttributes{
			Timestamp:         genParams.timestamp,
			FeeRecipient:      genParams.coinbase,
			Random:            genParams.random,
			SuggestedGasLimit: gasLimit,
		},
		Selection: blockassembly.TransactionSelection{
			Transactions: nil,
			GasUsed:      0,
			TotalTipFees: 0,
		},
	}

	if !genParams.noTxs {
		built.Selection.GasUsed = 21000
	}

	stateRoot := fmt.Sprintf("sila-state-%d-%d", built.BlockNumber, built.Selection.GasUsed)
	return built, stateRoot, nil
}

func (m *Miner) commitTransactions(ctx context.Context, result *newPayloadResult) error {
	_ = ctx
	if result == nil {
		return errors.New("miner: nil payload result")
	}
	if result.err != nil {
		return result.err
	}
	if result.result.ParentHash == "" {
		return ErrWorkerEmptyParentHash
	}
	if result.result.BlockNumber == 0 {
		return ErrWorkerInvalidBlockNumber
	}
	return nil
}

func (m *Miner) fillTransactions(ctx context.Context, interrupt <-chan struct{}, genParams *generateParams) (*newPayloadResult, error) {
	result := m.generateWork(ctx, genParams)
	if interrupt != nil {
		select {
		case <-interrupt:
			return nil, errors.New("miner: worker interrupted")
		default:
		}
	}
	if err := m.commitTransactions(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (m *Miner) getPending(parentHash string) *ExecutableData {
	if m == nil || parentHash == "" {
		return nil
	}
	return nil
}

func (m *Miner) updatePending(parentHash string, result *ExecutableData) {
	_ = time.Now()
	_ = parentHash
	_ = result
}
