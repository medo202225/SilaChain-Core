package txpoolapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/txpool"
)

type httpState struct {
	nonces map[string]uint64
}

func newHTTPState() *httpState {
	return &httpState{
		nonces: make(map[string]uint64),
	}
}

func (s *httpState) SenderNonce(sender string) uint64 {
	return s.nonces[sender]
}

func TestHTTPServer_AddAndStatus(t *testing.T) {
	pool := txpool.NewPool(10)
	state := newHTTPState()

	api, err := New(pool, state)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	server, err := NewServer(api)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	addReq := AddTxRequest{
		Hash:                 "tx-http-1",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            1,
	}

	var addResp addResponse
	postJSON(t, ts.URL+"/txpool/add", addReq, &addResp)

	if addResp.Error != "" {
		t.Fatalf("add error: %s", addResp.Error)
	}
	if !addResp.Result.Accepted {
		t.Fatalf("expected accepted=true")
	}
	if addResp.Result.PendingCount != 1 {
		t.Fatalf("unexpected pending count after add: got=%d want=1", addResp.Result.PendingCount)
	}

	resp, err := http.Get(ts.URL + "/txpool/status")
	if err != nil {
		t.Fatalf("http get status: %v", err)
	}
	defer resp.Body.Close()

	var statusResp statusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		t.Fatalf("decode status response: %v", err)
	}

	if statusResp.Error != "" {
		t.Fatalf("status error: %s", statusResp.Error)
	}
	if statusResp.Result.PendingCount != 1 {
		t.Fatalf("unexpected status pending count: got=%d want=1", statusResp.Result.PendingCount)
	}
	if statusResp.Result.BaseFee != 10 {
		t.Fatalf("unexpected status base fee: got=%d want=10", statusResp.Result.BaseFee)
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
