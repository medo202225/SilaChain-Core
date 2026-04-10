package runtime

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestChainLogs_AfterProduceBlock_ReturnsCanonicalLogs(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:10005",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xlogs-genesis",
			StateRoot: "0xlogs-state",
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
		Hash:                 "tx-logs-1",
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
		Timestamp:         9501,
		FeeRecipient:      "SILA_fee_recipient_logs",
		Random:            "SILA_random_logs",
		SuggestedGasLimit: 0,
	}); err != nil {
		t.Fatalf("produce block: %v", err)
	}

	txLogs, err := rt.ChainLogs("tx-logs-1")
	if err != nil {
		t.Fatalf("chain logs by tx: %v", err)
	}
	if !txLogs.Found {
		t.Fatalf("expected tx logs lookup to be found")
	}
	if len(txLogs.Logs) != 1 {
		t.Fatalf("unexpected tx logs count: got=%d want=1", len(txLogs.Logs))
	}

	blockByNumber, err := rt.ChainBlockByNumber(1)
	if err != nil {
		t.Fatalf("chain block by number: %v", err)
	}

	blockLogs, err := rt.ChainLogsByBlock(blockByNumber.Block.Hash)
	if err != nil {
		t.Fatalf("chain logs by block: %v", err)
	}
	if !blockLogs.Found {
		t.Fatalf("expected block logs lookup to be found")
	}
	if len(blockLogs.Logs) != 1 {
		t.Fatalf("unexpected block logs count: got=%d want=1", len(blockLogs.Logs))
	}

	blockLogsByNumber, err := rt.ChainLogsByBlockNumber(1)
	if err != nil {
		t.Fatalf("chain logs by block number: %v", err)
	}
	if !blockLogsByNumber.Found {
		t.Fatalf("expected block logs by number lookup to be found")
	}
	if len(blockLogsByNumber.Logs) != 1 {
		t.Fatalf("unexpected block logs by number count: got=%d want=1", len(blockLogsByNumber.Logs))
	}
}

func TestIntrospectionServer_ExposesLogsEndpoints(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:10006",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xlogs-http-genesis",
			StateRoot: "0xlogs-http-state",
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
		Hash:                 "tx-logs-http-1",
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
		Timestamp:         9502,
		FeeRecipient:      "SILA_fee_recipient_logs_http",
		Random:            "SILA_random_logs_http",
		SuggestedGasLimit: 0,
	}); err != nil {
		t.Fatalf("produce block: %v", err)
	}

	server, err := NewIntrospectionServer(rt)
	if err != nil {
		t.Fatalf("new introspection server: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/chain/logs?txHash=tx-logs-http-1", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("unexpected chain logs status code: got=%d want=200", rec.Code)
	}

	var decoded chainLogsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode logs response: %v", err)
	}
	if !decoded.Result.Found {
		t.Fatalf("expected http logs lookup to be found")
	}
	if len(decoded.Result.Logs) != 1 {
		t.Fatalf("unexpected http logs count: got=%d want=1", len(decoded.Result.Logs))
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/chain/logsByBlockNumber?number=1", nil)
	server.Handler().ServeHTTP(rec2, req2)

	if rec2.Code != 200 {
		t.Fatalf("unexpected chain logs by block number status code: got=%d want=200", rec2.Code)
	}

	var decoded2 chainLogsResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &decoded2); err != nil {
		t.Fatalf("decode logs-by-number response: %v", err)
	}
	if !decoded2.Result.Found {
		t.Fatalf("expected http logs by number lookup to be found")
	}
	if len(decoded2.Result.Logs) != 1 {
		t.Fatalf("unexpected http logs by number count: got=%d want=1", len(decoded2.Result.Logs))
	}
}
