package runtime

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/forkchoice"
)

func TestChainBlockTransactions_AfterProduceBlock(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9999",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xblocktxs-genesis",
			StateRoot: "0xblocktxs-state",
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
		Hash:                 "tx-blocktxs-1",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            1,
	}); err != nil {
		t.Fatalf("add tx: %v", err)
	}

	produced, err := rt.ProduceBlock(ProduceBlockRequest{
		Timestamp:         9201,
		FeeRecipient:      "SILA_fee_recipient_blocktxs",
		Random:            "SILA_random_blocktxs",
		SuggestedGasLimit: 0,
	})
	if err != nil {
		t.Fatalf("produce block: %v", err)
	}

	canonicalHead, ok := produced.CanonicalHead.(forkchoice.BlockRef)
	if !ok {
		t.Fatalf("unexpected canonical head type")
	}

	byHash, err := rt.ChainBlockTransactions(canonicalHead.Hash)
	if err != nil {
		t.Fatalf("chain block txs by hash: %v", err)
	}
	if !byHash.Found {
		t.Fatalf("expected block txs by hash to be found")
	}
	if len(byHash.Transactions) != 1 {
		t.Fatalf("unexpected tx count by hash: got=%d want=1", len(byHash.Transactions))
	}
	if byHash.Transactions[0].Hash != "tx-blocktxs-1" {
		t.Fatalf("unexpected tx hash by hash: got=%s want=tx-blocktxs-1", byHash.Transactions[0].Hash)
	}

	byNumber, err := rt.ChainBlockTransactionsByNumber(1)
	if err != nil {
		t.Fatalf("chain block txs by number: %v", err)
	}
	if !byNumber.Found {
		t.Fatalf("expected block txs by number to be found")
	}
	if len(byNumber.Transactions) != 1 {
		t.Fatalf("unexpected tx count by number: got=%d want=1", len(byNumber.Transactions))
	}
	if byNumber.Transactions[0].Hash != "tx-blocktxs-1" {
		t.Fatalf("unexpected tx hash by number: got=%s want=tx-blocktxs-1", byNumber.Transactions[0].Hash)
	}
}

func TestIntrospectionServer_ExposesBlockTransactionsEndpoints(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:10000",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xblocktxs-http-genesis",
			StateRoot: "0xblocktxs-http-state",
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
		Hash:                 "tx-blocktxs-http-1",
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
		Timestamp:         9202,
		FeeRecipient:      "SILA_fee_recipient_blocktxs_http",
		Random:            "SILA_random_blocktxs_http",
		SuggestedGasLimit: 0,
	}); err != nil {
		t.Fatalf("produce block: %v", err)
	}

	server, err := NewIntrospectionServer(rt)
	if err != nil {
		t.Fatalf("new introspection server: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/chain/blockTxsByNumber?number=1", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("unexpected block txs by number status code: got=%d want=200", rec.Code)
	}

	var decoded chainBlockTransactionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode block txs response: %v", err)
	}
	if !decoded.Result.Found {
		t.Fatalf("expected http block tx listing to be found")
	}
	if len(decoded.Result.Transactions) != 1 {
		t.Fatalf("unexpected http block tx count: got=%d want=1", len(decoded.Result.Transactions))
	}
	if decoded.Result.Transactions[0].Hash != "tx-blocktxs-http-1" {
		t.Fatalf("unexpected http tx hash: got=%s want=tx-blocktxs-http-1", decoded.Result.Transactions[0].Hash)
	}
}
