package rpc

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"silachain/internal/chain"
)

func newConcurrentStressBlockchain(t *testing.T) *chain.Blockchain {
	t.Helper()

	dataDir := filepath.Join(t.TempDir(), "rpc-concurrent-stress")
	bc, err := chain.NewBlockchain(dataDir, nil, 0)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	return bc
}

func newConcurrentStressServer(t *testing.T, bc *chain.Blockchain) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/chain/info", ChainInfoHandler(bc))
	mux.HandleFunc("/explorer/summary", ExplorerSummaryHandler(bc))
	mux.HandleFunc("/explorer/network", NetworkStatusHandler(bc))
	mux.HandleFunc("/logs/query", LogsQueryHandler(bc))
	mux.HandleFunc("/sila-rpc", SilaRPCHandler(bc))

	return httptest.NewServer(mux)
}

func TestRPCConcurrentStress_ReadEndpoints(t *testing.T) {
	bc := newConcurrentStressBlockchain(t)
	server := newConcurrentStressServer(t, bc)
	defer server.Close()

	type endpoint struct {
		name string
		path string
	}

	endpoints := []endpoint{
		{name: "chain_info", path: "/chain/info"},
		{name: "explorer_summary", path: "/explorer/summary"},
		{name: "explorer_network", path: "/explorer/network"},
		{name: "logs_query", path: "/logs/query"},
	}

	const workers = 20
	const iterationsPerWorker = 25

	var failures int64
	var wg sync.WaitGroup
	client := server.Client()

	for _, ep := range endpoints {
		ep := ep
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for i := 0; i < iterationsPerWorker; i++ {
					resp, err := client.Get(server.URL + ep.path)
					if err != nil {
						atomic.AddInt64(&failures, 1)
						continue
					}

					_, _ = io.ReadAll(resp.Body)
					_ = resp.Body.Close()

					if resp.StatusCode != http.StatusOK {
						atomic.AddInt64(&failures, 1)
					}
				}
			}()
		}
	}

	wg.Wait()

	if failures != 0 {
		t.Fatalf("expected zero failures, got %d", failures)
	}
}

func TestRPCConcurrentStress_SilaRPCChainID(t *testing.T) {
	bc := newConcurrentStressBlockchain(t)
	server := newConcurrentStressServer(t, bc)
	defer server.Close()

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

	const workers = 30
	const iterationsPerWorker = 20

	var failures int64
	var wg sync.WaitGroup
	client := server.Client()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; i < iterationsPerWorker; i++ {
				req, err := http.NewRequest(http.MethodPost, server.URL+"/sila-rpc", bytes.NewReader(raw))
				if err != nil {
					atomic.AddInt64(&failures, 1)
					continue
				}
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					atomic.AddInt64(&failures, 1)
					continue
				}

				body, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					_ = body
					atomic.AddInt64(&failures, 1)
				}
			}
		}()
	}

	wg.Wait()

	if failures != 0 {
		t.Fatalf("expected zero failures, got %d", failures)
	}
}

func TestRPCConcurrentStress_InvalidJSONStillRejected(t *testing.T) {
	bc := newConcurrentStressBlockchain(t)
	server := newConcurrentStressServer(t, bc)
	defer server.Close()

	const workers = 10
	const iterationsPerWorker = 15

	var badStatusCount int64
	var wg sync.WaitGroup
	client := server.Client()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; i < iterationsPerWorker; i++ {
				req, err := http.NewRequest(http.MethodPost, server.URL+"/sila-rpc", bytes.NewBufferString("{bad json"))
				if err != nil {
					atomic.AddInt64(&badStatusCount, 1)
					continue
				}
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					atomic.AddInt64(&badStatusCount, 1)
					continue
				}

				_, _ = io.ReadAll(resp.Body)
				_ = resp.Body.Close()

				if resp.StatusCode != http.StatusBadRequest {
					atomic.AddInt64(&badStatusCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	if badStatusCount != 0 {
		t.Fatalf("expected all invalid json requests to return 400, got %d failures", badStatusCount)
	}
}
