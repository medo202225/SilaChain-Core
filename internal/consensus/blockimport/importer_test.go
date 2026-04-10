package blockimport

import (
	"errors"
	"testing"

	"silachain/core"
	"silachain/internal/consensus/blockassembly"
)

type stubState struct {
	head blockassembly.Head
}

func (s *stubState) Head() blockassembly.Head {
	return s.head
}

func (s *stubState) SetHead(head blockassembly.Head) error {
	s.head = head
	return nil
}

func (s *stubState) SetSenderNonce(sender string, nonce uint64) error {
	return nil
}

type stubExecutor struct {
	result core.Result
	err    error
}

func (e *stubExecutor) Process(attrs blockassembly.PayloadAttributes) (core.Result, error) {
	if e.err != nil {
		return core.Result{}, e.err
	}
	return e.result, nil
}

func TestImport_SucceedsWithExpectedParentAndBlockNumber(t *testing.T) {
	state := &stubState{
		head: blockassembly.Head{
			Number:    12,
			Hash:      "0xhead12",
			StateRoot: "0xstate12",
			BaseFee:   10,
		},
	}

	executor := &stubExecutor{
		result: core.Result{
			BlockNumber:        13,
			BlockHash:          "sila-block-13-0xhead12-2",
			ParentHash:         "0xhead12",
			ExecutionStateRoot: "sila-state-13-2",
			BaseFee:            10,
			GasUsed:            42000,
			TxCount:            2,
		},
	}

	importer, err := New(state, executor)
	if err != nil {
		t.Fatalf("new importer: %v", err)
	}

	result, err := importer.Import(ImportRequest{
		ExpectedParentHash:  "0xhead12",
		ExpectedBlockNumber: 13,
		Attributes: blockassembly.PayloadAttributes{
			Timestamp:    1000,
			FeeRecipient: "SILA_fee_recipient_import",
			Random:       "SILA_random_import",
		},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	if !result.Imported {
		t.Fatalf("expected imported=true")
	}
	if result.AlreadyImported {
		t.Fatalf("expected alreadyImported=false")
	}
	if result.BlockNumber != 13 {
		t.Fatalf("unexpected block number: got=%d want=13", result.BlockNumber)
	}
	if result.ParentHash != "0xhead12" {
		t.Fatalf("unexpected parent hash: got=%s want=0xhead12", result.ParentHash)
	}
	if result.TxCount != 2 {
		t.Fatalf("unexpected tx count: got=%d want=2", result.TxCount)
	}
}

func TestImport_FailsOnParentHashMismatch(t *testing.T) {
	state := &stubState{
		head: blockassembly.Head{
			Number: 20,
			Hash:   "0xactual-parent",
		},
	}

	executor := &stubExecutor{
		result: core.Result{
			BlockNumber: 21,
			BlockHash:   "sila-block-21",
			ParentHash:  "0xactual-parent",
		},
	}

	importer, err := New(state, executor)
	if err != nil {
		t.Fatalf("new importer: %v", err)
	}

	_, err = importer.Import(ImportRequest{
		ExpectedParentHash:  "0xwrong-parent",
		ExpectedBlockNumber: 21,
		Attributes:          blockassembly.PayloadAttributes{},
	})
	if err == nil {
		t.Fatalf("expected parent hash mismatch error")
	}
	if !errors.Is(err, ErrParentHashMismatch) {
		t.Fatalf("expected ErrParentHashMismatch, got=%v", err)
	}
}

func TestImport_FailsOnBlockNumberMismatch(t *testing.T) {
	state := &stubState{
		head: blockassembly.Head{
			Number: 30,
			Hash:   "0xparent30",
		},
	}

	executor := &stubExecutor{
		result: core.Result{
			BlockNumber: 31,
			BlockHash:   "sila-block-31",
			ParentHash:  "0xparent30",
		},
	}

	importer, err := New(state, executor)
	if err != nil {
		t.Fatalf("new importer: %v", err)
	}

	_, err = importer.Import(ImportRequest{
		ExpectedParentHash:  "0xparent30",
		ExpectedBlockNumber: 32,
		Attributes:          blockassembly.PayloadAttributes{},
	})
	if err == nil {
		t.Fatalf("expected block number mismatch error")
	}
	if !errors.Is(err, ErrBlockNumberMismatch) {
		t.Fatalf("expected ErrBlockNumberMismatch, got=%v", err)
	}
}

func TestImport_RejectsDuplicateImportedBlockHash(t *testing.T) {
	state := &stubState{
		head: blockassembly.Head{
			Number: 40,
			Hash:   "0xparent40",
		},
	}

	executor := &stubExecutor{
		result: core.Result{
			BlockNumber:        41,
			BlockHash:          "sila-block-41-dup",
			ParentHash:         "0xparent40",
			ExecutionStateRoot: "sila-state-41-1",
			GasUsed:            21000,
			TxCount:            1,
		},
	}

	importer, err := New(state, executor)
	if err != nil {
		t.Fatalf("new importer: %v", err)
	}

	_, err = importer.Import(ImportRequest{
		ExpectedParentHash:  "0xparent40",
		ExpectedBlockNumber: 41,
		Attributes:          blockassembly.PayloadAttributes{},
	})
	if err != nil {
		t.Fatalf("first import should succeed: %v", err)
	}

	state.head = blockassembly.Head{
		Number: 40,
		Hash:   "0xparent40",
	}

	result, err := importer.Import(ImportRequest{
		ExpectedParentHash:  "0xparent40",
		ExpectedBlockNumber: 41,
		Attributes:          blockassembly.PayloadAttributes{},
	})
	if err == nil {
		t.Fatalf("expected duplicate import error")
	}
	if !errors.Is(err, ErrBlockAlreadyImported) {
		t.Fatalf("expected ErrBlockAlreadyImported, got=%v", err)
	}
	if !result.AlreadyImported {
		t.Fatalf("expected alreadyImported=true")
	}
	if result.Imported {
		t.Fatalf("expected imported=false")
	}
}
