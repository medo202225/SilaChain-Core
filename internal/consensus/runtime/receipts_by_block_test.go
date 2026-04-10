package runtime

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestChainReceiptsByBlock_AfterProduceBlock(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:10004",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xreceiptsblock-genesis",
			StateRoot: "0xreceiptsblock-state",
			BaseFee:   1,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	inputs := []struct {
		Hash                 string `json:"hash"`
		From                 string `json:"from"`
		Nonce                uint64 `json:"nonce"`
		GasLimit             uint64 `json:"gas_limit"`
		MaxFeePerGas         uint64 `json:"max_fee_per_gas"`
		MaxPriorityFeePerGas uint64 `json:"max_priority_fee_per_gas"`
		Timestamp            int64  `json:"timestamp"`
	}{
		{Hash: "tx-receiptsblock-1", From: "alice", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 20, MaxPriorityFeePerGas: 2, Timestamp: 1},
		{Hash: "tx-receiptsblock-2", From: "bob", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 30, MaxPriorityFeePerGas: 3, Timestamp: 2},
	}

	for _, tx := range inputs {
		if _, err := rt.txpoolAPI.Add(tx); err != nil {
			t.Fatalf("add tx %s: %v", tx.Hash, err)
		}
	}

	if _, err := rt.ProduceBlock(ProduceBlockRequest{
		Timestamp:         9401,
		FeeRecipient:      "SILA_fee_recipient_receiptsblock",
		Random:            "SILA_random_receiptsblock",
		SuggestedGasLimit: 0,
	}); err != nil {
		t.Fatalf("produce block: %v", err)
	}

	byNumber, err := rt.ChainReceiptsByBlockNumber(1)
	if err != nil {
		t.Fatalf("chain receipts by number: %v", err)
	}
	if !byNumber.Found {
		t.Fatalf("expected receipts by number to be found")
	}
	if len(byNumber.Receipts) != 2 {
		t.Fatalf("unexpected receipts count by number: got=%d want=2", len(byNumber.Receipts))
	}

	gotOrder := []string{byNumber.Receipts[0].TxHash, byNumber.Receipts[1].TxHash}
	wantOrder := []string{"tx-receiptsblock-2", "tx-receiptsblock-1"}

	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Fatalf("unexpected receipt order at index %d: got=%s want=%s full=%v", i, gotOrder[i], wantOrder[i], gotOrder)
		}
	}

	if byNumber.Receipts[0].GasUsed != 21000 {
		t.Fatalf("unexpected first receipt gas used: got=%d want=21000", byNumber.Receipts[0].GasUsed)
	}
	if byNumber.Receipts[1].GasUsed != 21000 {
		t.Fatalf("unexpected second receipt gas used: got=%d want=21000", byNumber.Receipts[1].GasUsed)
	}
	if len(byNumber.Receipts[0].Logs) != 1 {
		t.Fatalf("unexpected first receipt logs count: got=%d want=1", len(byNumber.Receipts[0].Logs))
	}
	if len(byNumber.Receipts[1].Logs) != 1 {
		t.Fatalf("unexpected second receipt logs count: got=%d want=1", len(byNumber.Receipts[1].Logs))
	}

	blockByNumber, err := rt.ChainBlockByNumber(1)
	if err != nil {
		t.Fatalf("chain block by number: %v", err)
	}
	byHash, err := rt.ChainReceiptsByBlock(blockByNumber.Block.Hash)
	if err != nil {
		t.Fatalf("chain receipts by hash: %v", err)
	}
	if !byHash.Found {
		t.Fatalf("expected receipts by hash to be found")
	}
	if len(byHash.Receipts) != 2 {
		t.Fatalf("unexpected receipts count by hash: got=%d want=2", len(byHash.Receipts))
	}

	gotHashOrder := []string{byHash.Receipts[0].TxHash, byHash.Receipts[1].TxHash}
	for i := range wantOrder {
		if gotHashOrder[i] != wantOrder[i] {
			t.Fatalf("unexpected hash receipt order at index %d: got=%s want=%s full=%v", i, gotHashOrder[i], wantOrder[i], gotHashOrder)
		}
	}

	if len(byHash.Receipts[0].Logs) != 1 {
		t.Fatalf("unexpected first hash receipt logs count: got=%d want=1", len(byHash.Receipts[0].Logs))
	}
}

func TestIntrospectionServer_ExposesReceiptsByBlockEndpoints(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:10008",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xreceiptsblock-http-genesis",
			StateRoot: "0xreceiptsblock-http-state",
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
		Hash:                 "tx-receiptsblock-http-1",
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
		Timestamp:         9402,
		FeeRecipient:      "SILA_fee_recipient_receiptsblock_http",
		Random:            "SILA_random_receiptsblock_http",
		SuggestedGasLimit: 0,
	}); err != nil {
		t.Fatalf("produce block: %v", err)
	}

	server, err := NewIntrospectionServer(rt)
	if err != nil {
		t.Fatalf("new introspection server: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/chain/receiptsByBlockNumber?number=1", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("unexpected receipts by block number status code: got=%d want=200", rec.Code)
	}

	var decoded chainReceiptsByBlockResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode receipts by block response: %v", err)
	}

	if !decoded.Result.Found {
		t.Fatalf("expected http receipts by block to be found")
	}
	if len(decoded.Result.Receipts) != 1 {
		t.Fatalf("unexpected http receipts count: got=%d want=1", len(decoded.Result.Receipts))
	}
	if decoded.Result.Receipts[0].TxHash != "tx-receiptsblock-http-1" {
		t.Fatalf("unexpected http receipt tx hash: got=%s want=tx-receiptsblock-http-1", decoded.Result.Receipts[0].TxHash)
	}
	if len(decoded.Result.Receipts[0].Logs) != 1 {
		t.Fatalf("unexpected http receipt logs count: got=%d want=1", len(decoded.Result.Receipts[0].Logs))
	}
	if decoded.Result.Receipts[0].Logs[0].Address != "alice" {
		t.Fatalf("unexpected http receipt log address: got=%s want=alice", decoded.Result.Receipts[0].Logs[0].Address)
	}
}
