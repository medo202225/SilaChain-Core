package forkchoice

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/blockimport"
)

type stubImporter struct {
	result blockimport.Result
	err    error
}

func (s *stubImporter) Import(req blockimport.ImportRequest) (blockimport.Result, error) {
	if s.err != nil {
		return blockimport.Result{}, s.err
	}
	return s.result, nil
}

func TestImportAndApply_AdvancesCanonicalHeadAfterSuccessfulImport(t *testing.T) {
	store, err := New(blockassembly.Head{
		Number:    9,
		Hash:      "0xhead9",
		StateRoot: "0xstate9",
		BaseFee:   10,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	manager, err := NewManager(&stubImporter{
		result: blockimport.Result{
			Imported:    true,
			BlockNumber: 10,
			BlockHash:   "0xblock10",
			ParentHash:  "0xhead9",
			StateRoot:   "0xstate10",
			GasUsed:     42000,
			TxCount:     2,
		},
	}, store)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	result, err := manager.ImportAndApply(blockimport.ImportRequest{
		ExpectedParentHash:  "0xhead9",
		ExpectedBlockNumber: 10,
		Attributes: blockassembly.PayloadAttributes{
			Timestamp:    1000,
			FeeRecipient: "SILA_fee_recipient_fc",
			Random:       "SILA_random_fc",
		},
	})
	if err != nil {
		t.Fatalf("import and apply: %v", err)
	}

	if !result.Import.Imported {
		t.Fatalf("expected import imported=true")
	}
	if !result.ForkChoice.Accepted {
		t.Fatalf("expected forkchoice accepted=true")
	}
	if !result.ForkChoice.CanonicalChanged {
		t.Fatalf("expected canonical head change")
	}
	if result.ForkChoice.CanonicalHead.Hash != "0xblock10" {
		t.Fatalf("unexpected canonical head hash: got=%s want=0xblock10", result.ForkChoice.CanonicalHead.Hash)
	}
}
