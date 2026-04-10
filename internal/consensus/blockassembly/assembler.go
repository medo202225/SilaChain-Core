package blockassembly

import (
	"errors"

	"silachain/internal/consensus/blockbuilder"
	"silachain/internal/consensus/txpool"
)

var (
	ErrNilStateProvider = errors.New("blockassembly: nil state provider")
	ErrNilPool          = errors.New("blockassembly: nil tx pool")
	ErrZeroGasLimit     = errors.New("blockassembly: zero gas limit")
)

type Head struct {
	Number    uint64
	Hash      string
	StateRoot string
	BaseFee   uint64
}

type StateProvider interface {
	Head() Head
}

type PayloadAttributes struct {
	Timestamp         uint64
	FeeRecipient      string
	Random            string
	SuggestedGasLimit uint64
}

type TransactionSelection struct {
	Transactions []txpool.Tx
	GasUsed      uint64
	TotalTipFees uint64
}

type Result struct {
	ParentNumber    uint64
	BlockNumber     uint64
	ParentHash      string
	ParentStateRoot string
	BaseFee         uint64
	GasLimit        uint64
	Attributes      PayloadAttributes
	Selection       TransactionSelection
}

type Assembler struct {
	stateProvider StateProvider
	pool          *txpool.Pool
	gasLimit      uint64
}

func New(stateProvider StateProvider, pool *txpool.Pool, gasLimit uint64) (*Assembler, error) {
	if stateProvider == nil {
		return nil, ErrNilStateProvider
	}
	if pool == nil {
		return nil, ErrNilPool
	}
	if gasLimit == 0 {
		return nil, ErrZeroGasLimit
	}

	return &Assembler{
		stateProvider: stateProvider,
		pool:          pool,
		gasLimit:      gasLimit,
	}, nil
}

func (a *Assembler) Assemble(attrs PayloadAttributes) (Result, error) {
	if a == nil || a.stateProvider == nil {
		return Result{}, ErrNilStateProvider
	}
	if a.pool == nil {
		return Result{}, ErrNilPool
	}
	if a.gasLimit == 0 {
		return Result{}, ErrZeroGasLimit
	}

	head := a.stateProvider.Head()

	if err := a.pool.SetBaseFee(head.BaseFee); err != nil {
		return Result{}, err
	}

	builder, err := blockbuilder.New(a.gasLimit)
	if err != nil {
		return Result{}, err
	}

	built, err := builder.Build(a.pool)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		ParentNumber:    head.Number,
		BlockNumber:     head.Number + 1,
		ParentHash:      head.Hash,
		ParentStateRoot: head.StateRoot,
		BaseFee:         head.BaseFee,
		GasLimit:        a.gasLimit,
		Attributes:      attrs,
		Selection: TransactionSelection{
			Transactions: built.Transactions,
			GasUsed:      built.GasUsed,
			TotalTipFees: built.TotalTipFees,
		},
	}

	if attrs.SuggestedGasLimit > 0 {
		result.GasLimit = attrs.SuggestedGasLimit
	}

	return result, nil
}
