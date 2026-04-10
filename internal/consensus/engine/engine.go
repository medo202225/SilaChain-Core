package engine

import (
	"errors"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/blockimport"
	"silachain/internal/consensus/forkchoice"
	"silachain/internal/consensus/payloadexecution"
	"silachain/internal/consensus/txpool"
)

var (
	ErrNilState     = errors.New("engine: nil state")
	ErrNilPool      = errors.New("engine: nil tx pool")
	ErrZeroGasLimit = errors.New("engine: zero gas limit")
)

type State interface {
	blockassembly.StateProvider
	payloadexecution.State
	SetHead(head blockassembly.Head) error
	SetSenderNonce(sender string, nonce uint64) error
}

type Engine struct {
	state    State
	pool     *txpool.Pool
	gasLimit uint64

	assembler *blockassembly.Assembler
	executor  *payloadexecution.Executor
	importer  *blockimport.Importer
	forkStore *forkchoice.Store
	manager   *forkchoice.Manager
}

type ProduceResult struct {
	ImportResult     blockimport.Result
	ForkChoiceResult forkchoice.ApplyResult
	CanonicalHead    forkchoice.BlockRef
}

func New(state State, pool *txpool.Pool, gasLimit uint64) (*Engine, error) {
	if state == nil {
		return nil, ErrNilState
	}
	if pool == nil {
		return nil, ErrNilPool
	}
	if gasLimit == 0 {
		return nil, ErrZeroGasLimit
	}

	assembler, err := blockassembly.New(state, pool, gasLimit)
	if err != nil {
		return nil, err
	}

	executor, err := payloadexecution.New(state, assembler)
	if err != nil {
		return nil, err
	}

	importer, err := blockimport.New(state, executor)
	if err != nil {
		return nil, err
	}

	store, err := forkchoice.New(state.Head())
	if err != nil {
		return nil, err
	}

	manager, err := forkchoice.NewManager(importer, store)
	if err != nil {
		return nil, err
	}

	return &Engine{
		state:     state,
		pool:      pool,
		gasLimit:  gasLimit,
		assembler: assembler,
		executor:  executor,
		importer:  importer,
		forkStore: store,
		manager:   manager,
	}, nil
}

func (e *Engine) ProduceBlock(attrs blockassembly.PayloadAttributes) (ProduceResult, error) {
	head := e.state.Head()

	result, err := e.manager.ImportAndApply(blockimport.ImportRequest{
		ExpectedParentHash:  head.Hash,
		ExpectedBlockNumber: head.Number + 1,
		Attributes:          attrs,
	})
	if err != nil {
		return ProduceResult{}, err
	}

	canonicalHead, err := e.forkStore.CanonicalHead()
	if err != nil {
		return ProduceResult{}, err
	}

	return ProduceResult{
		ImportResult:     result.Import,
		ForkChoiceResult: result.ForkChoice,
		CanonicalHead:    canonicalHead,
	}, nil
}

func (e *Engine) CanonicalHead() (forkchoice.BlockRef, error) {
	return e.forkStore.CanonicalHead()
}

func (e *Engine) GasLimit() uint64 {
	if e == nil {
		return 0
	}
	return e.gasLimit
}
