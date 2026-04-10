package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"silachain/internal/chain"
)

func newStressBlockchain(t *testing.T) *chain.Blockchain {
	t.Helper()

	dataDir := filepath.Join(t.TempDir(), "rpc-stress")
	bc, err := chain.NewBlockchain(dataDir, nil, 0)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	return bc
}

func TestRPCStress_ReadEndpointsRepeated(t *testing.T) {
	bc := newStressBlockchain(t)

	readTests := []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{"chain_info", ChainInfoHandler(bc), "/chain/info"},
		{"explorer_summary", ExplorerSummaryHandler(bc), "/explorer/summary"},
		{"network_status", NetworkStatusHandler(bc), "/explorer/network"},
		{"latest_block", LatestBlockHandler(bc), "/blocks/latest"},
		{"logs_query", LogsQueryHandler(bc), "/logs/query"},
	}

	for _, tc := range readTests {
		for i := 0; i < 50; i++ {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()
			tc.handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("%s iteration %d: expected 200, got %d, body=%s", tc.name, i, rr.Code, rr.Body.String())
			}
		}
	}
}

func TestRPCStress_SilaRPCCallRepeated(t *testing.T) {
	bc := newStressBlockchain(t)
	handler := SilaRPCHandler(bc)

	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "sila_chainId",
		"params":  []any{},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodPost, "/sila-rpc", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("iteration %d: expected 200, got %d, body=%s", i, rr.Code, rr.Body.String())
		}
	}
}

func TestRPCStress_InvalidJSONRejectedRepeated(t *testing.T) {
	bc := newStressBlockchain(t)
	handler := SilaRPCHandler(bc)

	for i := 0; i < 50; i++ {
		req := httptest.NewRequest(http.MethodPost, "/sila-rpc", bytes.NewBufferString("{bad json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("iteration %d: expected 400, got %d, body=%s", i, rr.Code, rr.Body.String())
		}
	}
}
