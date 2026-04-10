package runtime

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestProduceBlockServer_EndToEndHTTP(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9882",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xproduce-http-genesis",
			StateRoot: "0xproduce-http-state",
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
		Hash:                 "tx-produce-http-1",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            1,
	}); err != nil {
		t.Fatalf("add tx: %v", err)
	}

	server, err := NewProduceBlockServer(rt)
	if err != nil {
		t.Fatalf("new produce block server: %v", err)
	}

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	body := ProduceBlockRequest{
		Timestamp:         4001,
		FeeRecipient:      "SILA_fee_recipient_http",
		Random:            "SILA_random_http",
		SuggestedGasLimit: 0,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	resp, err := http.Post(ts.URL+"/engine/produceBlock", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("http post: %v", err)
	}
	defer resp.Body.Close()

	var decoded produceBlockResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if decoded.Error != "" {
		t.Fatalf("produce block error: %s", decoded.Error)
	}
	if decoded.Result.PayloadID == "" {
		t.Fatalf("expected non-empty payload id")
	}
	if decoded.Result.PayloadStatus.Status != "VALID" {
		t.Fatalf("unexpected payload status: got=%s want=VALID", decoded.Result.PayloadStatus.Status)
	}
	if decoded.Result.TxPoolPending != 0 {
		t.Fatalf("expected empty pool after produce block, got=%d", decoded.Result.TxPoolPending)
	}
}
