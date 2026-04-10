package engineapiserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/engine"
	"silachain/internal/consensus/engineapi"
	"silachain/internal/consensus/txpool"
)

type rpcTestState struct {
	head   blockassembly.Head
	nonces map[string]uint64
}

func newRPCTestState(head blockassembly.Head) *rpcTestState {
	return &rpcTestState{
		head:   head,
		nonces: make(map[string]uint64),
	}
}

func (s *rpcTestState) Head() blockassembly.Head {
	return s.head
}

func (s *rpcTestState) SetHead(head blockassembly.Head) error {
	s.head = head
	return nil
}

func (s *rpcTestState) SetSenderNonce(sender string, nonce uint64) error {
	s.nonces[sender] = nonce
	return nil
}

func (s *rpcTestState) SenderNonce(sender string) uint64 {
	return s.nonces[sender]
}

func TestHTTPServer_EndToEndBuildGetSubmitAndMetadata(t *testing.T) {
	head := blockassembly.Head{
		Number:    70,
		Hash:      "0xhead70",
		StateRoot: "0xstate70",
		BaseFee:   10,
	}

	state := newRPCTestState(head)
	state.SetSenderNonce("alice", 0)
	state.SetSenderNonce("bob", 0)

	pool := txpool.NewPool(10)

	if err := pool.SetSenderStateNonce("alice", 0); err != nil {
		t.Fatalf("set alice sender nonce: %v", err)
	}
	if err := pool.SetSenderStateNonce("bob", 0); err != nil {
		t.Fatalf("set bob sender nonce: %v", err)
	}

	for _, tx := range []txpool.Tx{
		{Hash: "alice-0", From: "alice", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 20, MaxPriorityFeePerGas: 2, Timestamp: 1},
		{Hash: "bob-0", From: "bob", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 100, MaxPriorityFeePerGas: 50, Timestamp: 1},
	} {
		if err := pool.Add(tx); err != nil {
			t.Fatalf("add tx %s: %v", tx.Hash, err)
		}
	}

	eng, err := engine.New(state, pool, 42000)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	svc, err := engineapi.NewBuilderServiceFromEngine(eng)
	if err != nil {
		t.Fatalf("new builder service from engine: %v", err)
	}

	server, err := New(svc)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	fcBody := forkchoiceUpdatedRequest{
		State: engineapi.ForkchoiceState{
			HeadBlockHash:      "0xhead70",
			SafeBlockHash:      "0xhead70",
			FinalizedBlockHash: "0xhead70",
		},
		PayloadAttributes: &blockassembly.PayloadAttributes{
			Timestamp:         7001,
			FeeRecipient:      "SILA_fee_recipient_rpc",
			Random:            "SILA_random_rpc",
			SuggestedGasLimit: 0,
		},
	}

	var fcResp forkchoiceUpdatedResponse
	postJSON(t, ts.URL+"/engine/forkchoiceUpdated", fcBody, &fcResp)
	if fcResp.Error != "" {
		t.Fatalf("forkchoiceUpdated error: %s", fcResp.Error)
	}
	if fcResp.Result.PayloadStatus.Status != engineapi.PayloadStatusValid {
		t.Fatalf("unexpected forkchoice status: got=%s want=%s", fcResp.Result.PayloadStatus.Status, engineapi.PayloadStatusValid)
	}
	if fcResp.Result.PayloadID == "" {
		t.Fatalf("expected non-empty payload id")
	}

	var getPayloadResp getPayloadResponse
	postJSON(t, ts.URL+"/engine/getPayload", getPayloadRequest{
		PayloadID: fcResp.Result.PayloadID,
	}, &getPayloadResp)

	if getPayloadResp.Error != "" {
		t.Fatalf("getPayload error: %s", getPayloadResp.Error)
	}
	if getPayloadResp.Result.BlockNumber != 71 {
		t.Fatalf("unexpected payload block number: got=%d want=71", getPayloadResp.Result.BlockNumber)
	}
	if getPayloadResp.Result.TxCount != 2 {
		t.Fatalf("unexpected payload tx count: got=%d want=2", getPayloadResp.Result.TxCount)
	}

	var newPayloadResp newPayloadResponse
	postJSON(t, ts.URL+"/engine/newPayload", newPayloadRequest{
		Payload: engineapi.PayloadEnvelope{
			BlockNumber: getPayloadResp.Result.BlockNumber,
			BlockHash:   getPayloadResp.Result.BlockHash,
			ParentHash:  getPayloadResp.Result.ParentHash,
			StateRoot:   getPayloadResp.Result.StateRoot,
		},
	}, &newPayloadResp)

	if newPayloadResp.Error != "" {
		t.Fatalf("newPayload error: %s", newPayloadResp.Error)
	}
	if newPayloadResp.Status.Status != engineapi.PayloadStatusValid {
		t.Fatalf("unexpected newPayload status: got=%s want=%s", newPayloadResp.Status.Status, engineapi.PayloadStatusValid)
	}

	var finalFCResp forkchoiceUpdatedResponse
	postJSON(t, ts.URL+"/engine/forkchoiceUpdated", forkchoiceUpdatedRequest{
		State: engineapi.ForkchoiceState{
			HeadBlockHash:      getPayloadResp.Result.BlockHash,
			SafeBlockHash:      getPayloadResp.Result.BlockHash,
			FinalizedBlockHash: "0xhead70",
		},
	}, &finalFCResp)

	if finalFCResp.Error != "" {
		t.Fatalf("final forkchoiceUpdated error: %s", finalFCResp.Error)
	}
	if finalFCResp.Result.CanonicalHead.Hash == "" {
		t.Fatalf("expected non-empty canonical head hash")
	}
	if finalFCResp.Result.CanonicalHead.Hash != getPayloadResp.Result.BlockHash {
		t.Fatalf("unexpected canonical head hash: got=%s want=%s", finalFCResp.Result.CanonicalHead.Hash, getPayloadResp.Result.BlockHash)
	}
	if finalFCResp.Result.CanonicalHead.Number != 71 {
		t.Fatalf("unexpected canonical head number: got=%d want=71", finalFCResp.Result.CanonicalHead.Number)
	}

	var metaResp getPayloadMetadataResponse
	postJSON(t, ts.URL+"/engine/getPayloadMetadata", getPayloadMetadataRequest{
		PayloadID: fcResp.Result.PayloadID,
	}, &metaResp)

	if metaResp.Error != "" {
		t.Fatalf("getPayloadMetadata error: %s", metaResp.Error)
	}
	if !metaResp.Result.SubmittedToNewPayload {
		t.Fatalf("expected submitted=true in metadata")
	}
	if !metaResp.Result.Canonical {
		t.Fatalf("expected canonical=true in metadata")
	}
	if metaResp.Result.LatestStatus != "CANONICAL" {
		t.Fatalf("unexpected metadata status: got=%s want=CANONICAL", metaResp.Result.LatestStatus)
	}
}

func TestHTTPServer_RejectsUnsupportedMethod(t *testing.T) {
	head := blockassembly.Head{
		Number:    0,
		Hash:      "0xgenesis",
		StateRoot: "0xstate0",
		BaseFee:   1,
	}

	state := newRPCTestState(head)
	pool := txpool.NewPool(1)

	eng, err := engine.New(state, pool, 30000000)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	svc, err := engineapi.NewBuilderServiceFromEngine(eng)
	if err != nil {
		t.Fatalf("new builder service from engine: %v", err)
	}

	server, err := New(svc)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/engine/getPayload", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status code: got=%d want=%d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func postJSON(t *testing.T, url string, body any, out any) {
	t.Helper()

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("http post: %v", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
