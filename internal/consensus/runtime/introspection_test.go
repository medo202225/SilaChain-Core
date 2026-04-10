package runtime

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestChainIntrospection_AfterProduceBlock(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9991",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xintrospection-genesis",
			StateRoot: "0xintrospection-state",
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
		Hash:                 "tx-introspect-1",
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
		Timestamp:         6001,
		FeeRecipient:      "SILA_fee_recipient_introspection",
		Random:            "SILA_random_introspection",
		SuggestedGasLimit: 0,
	})
	if err != nil {
		t.Fatalf("produce block: %v", err)
	}

	headResult, err := rt.ChainHead()
	if err != nil {
		t.Fatalf("chain head: %v", err)
	}
	if headResult.Head.Number != 1 {
		t.Fatalf("unexpected head number: got=%d want=1", headResult.Head.Number)
	}

	forkchoiceResult, err := rt.ChainForkchoice()
	if err != nil {
		t.Fatalf("chain forkchoice: %v", err)
	}
	if forkchoiceResult.CanonicalHead.Number != 1 {
		t.Fatalf("unexpected canonical head number: got=%d want=1", forkchoiceResult.CanonicalHead.Number)
	}
	if forkchoiceResult.SafeHead.Number != 1 {
		t.Fatalf("unexpected safe head number: got=%d want=1", forkchoiceResult.SafeHead.Number)
	}
	if forkchoiceResult.FinalizedHead.Number != 0 {
		t.Fatalf("unexpected finalized head number: got=%d want=0", forkchoiceResult.FinalizedHead.Number)
	}

	blockResult, err := rt.ChainBlock(headResult.Head.Hash)
	if err != nil {
		t.Fatalf("chain block: %v", err)
	}
	if !blockResult.Found {
		t.Fatalf("expected block to be found")
	}
	if blockResult.Block.Hash != headResult.Head.Hash {
		t.Fatalf("unexpected block hash: got=%s want=%s", blockResult.Block.Hash, headResult.Head.Hash)
	}

	blocksResult, err := rt.ChainBlocks(10)
	if err != nil {
		t.Fatalf("chain blocks: %v", err)
	}
	if len(blocksResult.Blocks) != 2 {
		t.Fatalf("unexpected canonical blocks length: got=%d want=2", len(blocksResult.Blocks))
	}
	if blocksResult.Blocks[0].Number != 1 {
		t.Fatalf("unexpected latest canonical block number: got=%d want=1", blocksResult.Blocks[0].Number)
	}
	if blocksResult.Blocks[1].Number != 0 {
		t.Fatalf("unexpected previous canonical block number: got=%d want=0", blocksResult.Blocks[1].Number)
	}

	byNumberResult, err := rt.ChainBlockByNumber(1)
	if err != nil {
		t.Fatalf("chain block by number: %v", err)
	}
	if !byNumberResult.Found {
		t.Fatalf("expected block by number to be found")
	}
	if byNumberResult.Block.Number != 1 {
		t.Fatalf("unexpected block by number: got=%d want=1", byNumberResult.Block.Number)
	}

	if produced.TxPoolPending != 0 {
		t.Fatalf("expected empty txpool after produced block, got=%d", produced.TxPoolPending)
	}
}

func TestIntrospectionServer_ExposesChainEndpoints(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9992",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xintrospection-http-genesis",
			StateRoot: "0xintrospection-http-state",
			BaseFee:   1,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	server, err := NewIntrospectionServer(rt)
	if err != nil {
		t.Fatalf("new introspection server: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/chain/head", nil)
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("unexpected chain head status code: got=%d want=200", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/chain/blocks?limit=5", nil)
	server.Handler().ServeHTTP(rec2, req2)

	if rec2.Code != 200 {
		t.Fatalf("unexpected chain blocks status code: got=%d want=200", rec2.Code)
	}

	var decoded chainBlocksResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode chain blocks response: %v", err)
	}
	if len(decoded.Result.Blocks) != 1 {
		t.Fatalf("unexpected genesis-only chain blocks length: got=%d want=1", len(decoded.Result.Blocks))
	}
	if decoded.Result.Blocks[0].Number != 0 {
		t.Fatalf("unexpected genesis block number: got=%d want=0", decoded.Result.Blocks[0].Number)
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/chain/blockByNumber?number=0", nil)
	server.Handler().ServeHTTP(rec3, req3)

	if rec3.Code != 200 {
		t.Fatalf("unexpected chain blockByNumber status code: got=%d want=200", rec3.Code)
	}
}
