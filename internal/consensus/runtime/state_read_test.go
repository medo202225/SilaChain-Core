package runtime

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestStateAccount_AfterProduceBlock(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9997",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xstateaccount-genesis",
			StateRoot: "0xstateaccount-state",
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
		Hash:                 "tx-state-1",
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
		Timestamp:         9101,
		FeeRecipient:      "SILA_fee_recipient_state",
		Random:            "SILA_random_state",
		SuggestedGasLimit: 0,
	}); err != nil {
		t.Fatalf("produce block: %v", err)
	}

	result, err := rt.StateAccount("alice")
	if err != nil {
		t.Fatalf("state account: %v", err)
	}

	if !result.Found {
		t.Fatalf("expected account to be found")
	}
	if result.Nonce != 1 {
		t.Fatalf("unexpected account nonce: got=%d want=1", result.Nonce)
	}
	if result.Address != "alice" {
		t.Fatalf("unexpected account address: got=%s want=alice", result.Address)
	}
}

func TestIntrospectionServer_ExposesStateAccountEndpoint(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9998",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xstateaccount-http-genesis",
			StateRoot: "0xstateaccount-http-state",
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
		Hash:                 "tx-state-http-1",
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
		Timestamp:         9102,
		FeeRecipient:      "SILA_fee_recipient_state_http",
		Random:            "SILA_random_state_http",
		SuggestedGasLimit: 0,
	}); err != nil {
		t.Fatalf("produce block: %v", err)
	}

	server, err := NewIntrospectionServer(rt)
	if err != nil {
		t.Fatalf("new introspection server: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/state/account?address=alice", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("unexpected state account status code: got=%d want=200", rec.Code)
	}

	var decoded stateAccountResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode state account response: %v", err)
	}
	if !decoded.Result.Found {
		t.Fatalf("expected http account lookup to be found")
	}
	if decoded.Result.Nonce != 1 {
		t.Fatalf("unexpected http account nonce: got=%d want=1", decoded.Result.Nonce)
	}
}
