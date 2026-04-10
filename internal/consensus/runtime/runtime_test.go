package runtime

import (
	"context"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestNewRuntime_BuildsRealConsensusEngineStack(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9551",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xruntime-genesis",
			StateRoot: "0xruntime-state",
			BaseFee:   1,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	if rt.State() == nil {
		t.Fatalf("expected non-nil state")
	}
	if rt.Pool() == nil {
		t.Fatalf("expected non-nil pool")
	}
	if rt.Engine() == nil {
		t.Fatalf("expected non-nil engine")
	}
	if rt.API() == nil {
		t.Fatalf("expected non-nil api")
	}
	if rt.HTTPServer() == nil {
		t.Fatalf("expected non-nil http server")
	}
	if rt.Config().ListenAddress != "127.0.0.1:9551" {
		t.Fatalf("unexpected listen address: got=%s want=127.0.0.1:9551", rt.Config().ListenAddress)
	}
}

func TestRuntime_HTTPServer_ExposesHealthEndpoint(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9552",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xruntime-genesis-2",
			StateRoot: "0xruntime-state-2",
			BaseFee:   1,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	rt.HTTPServer().Handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("unexpected health status code: got=%d want=200", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatalf("expected non-empty health body")
	}
}

func TestRuntime_HTTPServer_ExposesTxPoolStatusEndpoint(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9554",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xruntime-genesis-4",
			StateRoot: "0xruntime-state-4",
			BaseFee:   7,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/txpool/status", nil)
	rt.HTTPServer().Handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("unexpected txpool status code: got=%d want=200", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatalf("expected non-empty txpool status body")
	}
}

func TestRuntime_Shutdown_WorksWithoutListen(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:9553",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xruntime-genesis-3",
			StateRoot: "0xruntime-state-3",
			BaseFee:   1,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}
