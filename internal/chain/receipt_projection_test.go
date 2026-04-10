package chain

import (
	"testing"

	"silachain/internal/core/state"
	pkgtypes "silachain/pkg/types"
)

func TestBuildReceiptsFromExecutionResult_ProjectsCanonicalFields(t *testing.T) {
	execResult := state.SuccessResult(
		pkgtypes.Hash("tx-hash"),
		pkgtypes.Address("from"),
		pkgtypes.Address("to"),
		10,
		2,
		12,
		21000,
		1,
		nil,
		"ret",
		"",
		"",
		123,
	)

	chainReceipt, blockReceipt := buildReceiptsFromExecutionResult(
		execResult,
		pkgtypes.Hash("block-hash"),
		7,
		0,
		2,
		21000,
		21000,
	)

	if chainReceipt.TxHash != blockReceipt.TxHash {
		t.Fatalf("expected matching tx hash")
	}
	if chainReceipt.CumulativeGasUsed != blockReceipt.CumulativeGasUsed {
		t.Fatalf("expected matching cumulative gas")
	}
	if chainReceipt.ReturnData != blockReceipt.ReturnData {
		t.Fatalf("expected matching return data")
	}
	if chainReceipt.BlockHash != "block-hash" || blockReceipt.BlockHash != "block-hash" {
		t.Fatalf("expected anchored block hash")
	}
	if chainReceipt.GasPrice != 2 {
		t.Fatalf("expected gas price on chain receipt")
	}
}
