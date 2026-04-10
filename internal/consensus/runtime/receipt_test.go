package runtime

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestChainReceipt_AfterProduceBlock(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:10003",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xreceipt-genesis",
			StateRoot: "0xreceipt-state",
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
		Hash:                 "tx-receipt-1",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            1,
	}); err != nil {
		t.Fatalf("add tx: %v", err)
	}

	if _, err := rt.ProduceBlock(ProduceBlockRequest{
		Timestamp:         9301,
		FeeRecipient:      "SILA_fee_recipient_receipt",
		Random:            "SILA_random_receipt",
		SuggestedGasLimit: 0,
	}); err != nil {
		t.Fatalf("produce block: %v", err)
	}

	result, err := rt.ChainReceipt("tx-receipt-1")
	if err != nil {
		t.Fatalf("chain receipt: %v", err)
	}

	if !result.Found {
		t.Fatalf("expected receipt to be found")
	}
	if result.TxHash != "tx-receipt-1" {
		t.Fatalf("unexpected receipt tx hash: got=%s want=tx-receipt-1", result.TxHash)
	}
	if result.BlockNumber != 1 {
		t.Fatalf("unexpected receipt block number: got=%d want=1", result.BlockNumber)
	}
	if result.GasUsed != 21000 {
		t.Fatalf("unexpected receipt gas used: got=%d want=21000", result.GasUsed)
	}
	if len(result.Logs) != 1 {
		t.Fatalf("unexpected receipt logs count: got=%d want=1", len(result.Logs))
	}
	if result.Logs[0].Address != "alice" {
		t.Fatalf("unexpected receipt log address: got=%s want=alice", result.Logs[0].Address)
	}
	if len(result.Logs[0].Topics) != 1 || result.Logs[0].Topics[0] != "0xsila_tx_included" {
		t.Fatalf("unexpected receipt log topics: got=%v", result.Logs[0].Topics)
	}
}

func TestIntrospectionServer_ExposesReceiptEndpoint(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:10007",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xreceipt-http-genesis",
			StateRoot: "0xreceipt-http-state",
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
		Hash:                 "tx-receipt-http-1",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            1,
	}); err != nil {
		t.Fatalf("add tx: %v", err)
	}

	if _, err := rt.ProduceBlock(ProduceBlockRequest{
		Timestamp:         9302,
		FeeRecipient:      "SILA_fee_recipient_receipt_http",
		Random:            "SILA_random_receipt_http",
		SuggestedGasLimit: 0,
	}); err != nil {
		t.Fatalf("produce block: %v", err)
	}

	server, err := NewIntrospectionServer(rt)
	if err != nil {
		t.Fatalf("new introspection server: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/chain/receipt?txHash=tx-receipt-http-1", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("unexpected receipt status code: got=%d want=200", rec.Code)
	}

	var decoded chainReceiptResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode receipt response: %v", err)
	}

	if !decoded.Result.Found {
		t.Fatalf("expected http receipt lookup to be found")
	}
	if decoded.Result.GasUsed != 21000 {
		t.Fatalf("unexpected http receipt gas used: got=%d want=21000", decoded.Result.GasUsed)
	}
	if len(decoded.Result.Logs) != 1 {
		t.Fatalf("unexpected http receipt logs count: got=%d want=1", len(decoded.Result.Logs))
	}
	if decoded.Result.Logs[0].Address != "alice" {
		t.Fatalf("unexpected http receipt log address: got=%s want=alice", decoded.Result.Logs[0].Address)
	}
}
