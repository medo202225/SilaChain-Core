package runtime

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestChainTransaction_AfterProduceBlock(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9995",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xtxlookup-genesis",
			StateRoot: "0xtxlookup-state",
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
		Hash:                 "tx-lookup-1",
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
		Timestamp:         9001,
		FeeRecipient:      "SILA_fee_recipient_txlookup",
		Random:            "SILA_random_txlookup",
		SuggestedGasLimit: 0,
	}); err != nil {
		t.Fatalf("produce block: %v", err)
	}

	result, err := rt.ChainTransaction("tx-lookup-1")
	if err != nil {
		t.Fatalf("chain transaction: %v", err)
	}

	if !result.Found {
		t.Fatalf("expected transaction to be found")
	}
	if result.BlockNumber != 1 {
		t.Fatalf("unexpected block number: got=%d want=1", result.BlockNumber)
	}
	if result.Transaction.Hash != "tx-lookup-1" {
		t.Fatalf("unexpected tx hash: got=%s want=tx-lookup-1", result.Transaction.Hash)
	}
	if result.Transaction.From != "alice" {
		t.Fatalf("unexpected tx from: got=%s want=alice", result.Transaction.From)
	}
}

func TestIntrospectionServer_ExposesChainTransactionEndpoint(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9996",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xtxlookup-http-genesis",
			StateRoot: "0xtxlookup-http-state",
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
		Hash:                 "tx-lookup-http-1",
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
		Timestamp:         9002,
		FeeRecipient:      "SILA_fee_recipient_txlookup_http",
		Random:            "SILA_random_txlookup_http",
		SuggestedGasLimit: 0,
	}); err != nil {
		t.Fatalf("produce block: %v", err)
	}

	server, err := NewIntrospectionServer(rt)
	if err != nil {
		t.Fatalf("new introspection server: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/chain/tx?hash=tx-lookup-http-1", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("unexpected chain tx status code: got=%d want=200", rec.Code)
	}

	var decoded chainTransactionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode chain tx response: %v", err)
	}
	if !decoded.Result.Found {
		t.Fatalf("expected http transaction lookup to be found")
	}
	if decoded.Result.BlockNumber != 1 {
		t.Fatalf("unexpected http transaction block number: got=%d want=1", decoded.Result.BlockNumber)
	}
}
