package runtime

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestProduceBlock_EndToEndSingleCall(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9881",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xproduce-genesis",
			StateRoot: "0xproduce-state",
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
		Hash:                 "tx-produce-1",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            1,
	}); err != nil {
		t.Fatalf("add tx: %v", err)
	}

	result, err := rt.ProduceBlock(ProduceBlockRequest{
		Timestamp:         3001,
		FeeRecipient:      "SILA_fee_recipient_produce",
		Random:            "SILA_random_produce",
		SuggestedGasLimit: 0,
	})
	if err != nil {
		t.Fatalf("produce block: %v", err)
	}

	if result.PayloadID == "" {
		t.Fatalf("expected non-empty payload id")
	}
	if result.PayloadStatus.Status != "VALID" {
		t.Fatalf("unexpected payload status: got=%s want=VALID", result.PayloadStatus.Status)
	}
	if result.TxPoolPending != 0 {
		t.Fatalf("expected empty txpool after produce block, got=%d", result.TxPoolPending)
	}

	head := rt.state.Head()
	if head.Number != 1 {
		t.Fatalf("unexpected state head number: got=%d want=1", head.Number)
	}
	if head.Hash == "" {
		t.Fatalf("expected non-empty state head hash")
	}
}
