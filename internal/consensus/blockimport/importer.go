package blockimport

import (
	"errors"
	"fmt"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/payloadexecution"
)

var (
	ErrNilState             = errors.New("blockimport: nil state")
	ErrNilExecutor          = errors.New("blockimport: nil executor")
	ErrEmptyExpectedParent  = errors.New("blockimport: empty expected parent hash")
	ErrParentHashMismatch   = errors.New("blockimport: parent hash mismatch")
	ErrBlockNumberMismatch  = errors.New("blockimport: block number mismatch")
	ErrBlockAlreadyImported = errors.New("blockimport: block already imported")
)

type State interface {
	Head() blockassembly.Head
	SetHead(head blockassembly.Head) error
	SetSenderNonce(sender string, nonce uint64) error
}

type Executor interface {
	Execute(attrs blockassembly.PayloadAttributes) (payloadexecution.Result, error)
}

type ImportRequest struct {
	ExpectedParentHash  string
	ExpectedBlockNumber uint64
	Attributes          blockassembly.PayloadAttributes
}

type Result struct {
	Imported        bool
	BlockNumber     uint64
	BlockHash       string
	ParentHash      string
	StateRoot       string
	GasUsed         uint64
	TxCount         int
	AlreadyImported bool
}

type Importer struct {
	state          State
	executor       Executor
	importedBlocks map[string]struct{}
}

func New(state State, executor Executor) (*Importer, error) {
	if state == nil {
		return nil, ErrNilState
	}
	if executor == nil {
		return nil, ErrNilExecutor
	}

	return &Importer{
		state:          state,
		executor:       executor,
		importedBlocks: make(map[string]struct{}),
	}, nil
}

func (i *Importer) Import(req ImportRequest) (Result, error) {
	if i == nil || i.state == nil {
		return Result{}, ErrNilState
	}
	if i.executor == nil {
		return Result{}, ErrNilExecutor
	}
	if req.ExpectedParentHash == "" {
		return Result{}, ErrEmptyExpectedParent
	}

	head := i.state.Head()

	if head.Hash != req.ExpectedParentHash {
		return Result{}, fmt.Errorf(
			"%w: expected=%s actual=%s",
			ErrParentHashMismatch,
			req.ExpectedParentHash,
			head.Hash,
		)
	}

	expectedBlockNumber := head.Number + 1
	if req.ExpectedBlockNumber != expectedBlockNumber {
		return Result{}, fmt.Errorf(
			"%w: expected=%d actual=%d",
			ErrBlockNumberMismatch,
			expectedBlockNumber,
			req.ExpectedBlockNumber,
		)
	}

	executed, err := i.executor.Execute(req.Attributes)
	if err != nil {
		return Result{}, err
	}

	if executed.ParentHash != req.ExpectedParentHash {
		return Result{}, fmt.Errorf(
			"%w: request=%s executed=%s",
			ErrParentHashMismatch,
			req.ExpectedParentHash,
			executed.ParentHash,
		)
	}

	if executed.BlockNumber != req.ExpectedBlockNumber {
		return Result{}, fmt.Errorf(
			"%w: request=%d executed=%d",
			ErrBlockNumberMismatch,
			req.ExpectedBlockNumber,
			executed.BlockNumber,
		)
	}

	if _, exists := i.importedBlocks[executed.BlockHash]; exists {
		return Result{
			Imported:        false,
			BlockNumber:     executed.BlockNumber,
			BlockHash:       executed.BlockHash,
			ParentHash:      executed.ParentHash,
			StateRoot:       executed.ExecutionStateRoot,
			GasUsed:         executed.GasUsed,
			TxCount:         executed.TxCount,
			AlreadyImported: true,
		}, ErrBlockAlreadyImported
	}

	for _, receipt := range executed.Receipts {
		if err := i.state.SetSenderNonce(receipt.From, receipt.Nonce+1); err != nil {
			return Result{}, err
		}
	}

	if err := i.state.SetHead(blockassembly.Head{
		Number:    executed.BlockNumber,
		Hash:      executed.BlockHash,
		StateRoot: executed.ExecutionStateRoot,
		BaseFee:   executed.BaseFee,
	}); err != nil {
		return Result{}, err
	}

	i.importedBlocks[executed.BlockHash] = struct{}{}

	return Result{
		Imported:        true,
		BlockNumber:     executed.BlockNumber,
		BlockHash:       executed.BlockHash,
		ParentHash:      executed.ParentHash,
		StateRoot:       executed.ExecutionStateRoot,
		GasUsed:         executed.GasUsed,
		TxCount:         executed.TxCount,
		AlreadyImported: false,
	}, nil
}
