package runtime

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/engineapi"
)

func TestPruneCanonicalPayload_RemovesImportedTransactionsFromPool(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9661",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xcleanup-genesis",
			StateRoot: "0xcleanup-state",
			BaseFee:   1,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	if err := rt.state.SetSenderNonce("alice", 0); err != nil {
		t.Fatalf("set alice nonce: %v", err)
	}

	_, err = rt.txpoolAPI.Add(struct {
		Hash                 string `json:"hash"`
		From                 string `json:"from"`
		Nonce                uint64 `json:"nonce"`
		GasLimit             uint64 `json:"gas_limit"`
		MaxFeePerGas         uint64 `json:"max_fee_per_gas"`
		MaxPriorityFeePerGas uint64 `json:"max_priority_fee_per_gas"`
		Timestamp            int64  `json:"timestamp"`
	}{
		Hash:                 "tx-cleanup-1",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            1,
	})
	if err != nil {
		t.Fatalf("add tx: %v", err)
	}

	buildResult, err := rt.api.ForkchoiceUpdatedWithAttributes(
		engineapi.ForkchoiceState{
			HeadBlockHash:      "0xcleanup-genesis",
			SafeBlockHash:      "0xcleanup-genesis",
			FinalizedBlockHash: "0xcleanup-genesis",
		},
		&blockassembly.PayloadAttributes{
			Timestamp:         1,
			FeeRecipient:      "SILA_fee_recipient_cleanup",
			Random:            "SILA_random_cleanup",
			SuggestedGasLimit: 0,
		},
	)
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	payload, err := rt.api.GetPayload(buildResult.PayloadID)
	if err != nil {
		t.Fatalf("get payload: %v", err)
	}

	if _, err := rt.api.NewPayload(engineapi.PayloadEnvelope{
		BlockNumber: payload.BlockNumber,
		BlockHash:   payload.BlockHash,
		ParentHash:  payload.ParentHash,
		StateRoot:   payload.StateRoot,
	}); err != nil {
		t.Fatalf("new payload: %v", err)
	}

	if _, err := rt.api.ForkchoiceUpdated(engineapi.ForkchoiceState{
		HeadBlockHash:      payload.BlockHash,
		SafeBlockHash:      payload.BlockHash,
		FinalizedBlockHash: "0xcleanup-genesis",
	}); err != nil {
		t.Fatalf("final forkchoice: %v", err)
	}

	if rt.pool.PendingCount() != 1 {
		t.Fatalf("expected pending tx before prune, got=%d", rt.pool.PendingCount())
	}

	if err := rt.PruneCanonicalPayload(buildResult.PayloadID); err != nil {
		t.Fatalf("prune canonical payload: %v", err)
	}

	if rt.pool.PendingCount() != 0 {
		t.Fatalf("expected empty pool after prune, got=%d", rt.pool.PendingCount())
	}
	if rt.pool.SenderStateNonce("alice") != 1 {
		t.Fatalf("unexpected alice pool sender nonce after prune: got=%d want=1", rt.pool.SenderStateNonce("alice"))
	}
}
