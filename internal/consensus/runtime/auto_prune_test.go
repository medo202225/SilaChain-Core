package runtime

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/engineapi"
)

func TestAPIAutoPrune_RemovesCanonicalTransactionsImmediately(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9771",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xautoprune-genesis",
			StateRoot: "0xautoprune-state",
			BaseFee:   1,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	if _, err := rt.txpoolAPI.Add(struct {
		Hash                 string `json:"hash"`
		From                 string `json:"from"`
		Nonce                uint64 `json:"nonce"`
		GasLimit             uint64 `json:"gas_limit"`
		MaxFeePerGas         uint64 `json:"max_fee_per_gas"`
		MaxPriorityFeePerGas uint64 `json:"max_priority_fee_per_gas"`
		Timestamp            int64  `json:"timestamp"`
	}{
		Hash:                 "tx-autoprune-1",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            1,
	}); err != nil {
		t.Fatalf("add tx: %v", err)
	}

	buildResult, err := rt.apiService.ForkchoiceUpdatedWithAttributes(
		engineapi.ForkchoiceState{
			HeadBlockHash:      "0xautoprune-genesis",
			SafeBlockHash:      "0xautoprune-genesis",
			FinalizedBlockHash: "0xautoprune-genesis",
		},
		&blockassembly.PayloadAttributes{
			Timestamp:         1,
			FeeRecipient:      "SILA_fee_recipient_autoprune",
			Random:            "SILA_random_autoprune",
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

	if _, err := rt.apiService.NewPayload(engineapi.PayloadEnvelope{
		BlockNumber: payload.BlockNumber,
		BlockHash:   payload.BlockHash,
		ParentHash:  payload.ParentHash,
		StateRoot:   payload.StateRoot,
	}); err != nil {
		t.Fatalf("new payload: %v", err)
	}

	if rt.pool.PendingCount() != 1 {
		t.Fatalf("expected pending tx before canonical forkchoice, got=%d", rt.pool.PendingCount())
	}

	if _, err := rt.apiService.ForkchoiceUpdated(engineapi.ForkchoiceState{
		HeadBlockHash:      payload.BlockHash,
		SafeBlockHash:      payload.BlockHash,
		FinalizedBlockHash: "0xautoprune-genesis",
	}); err != nil {
		t.Fatalf("canonical forkchoice: %v", err)
	}

	if rt.pool.PendingCount() != 0 {
		t.Fatalf("expected empty pool after auto prune, got=%d", rt.pool.PendingCount())
	}
	if rt.pool.SenderStateNonce("alice") != 1 {
		t.Fatalf("unexpected alice nonce after auto prune: got=%d want=1", rt.pool.SenderStateNonce("alice"))
	}
}
