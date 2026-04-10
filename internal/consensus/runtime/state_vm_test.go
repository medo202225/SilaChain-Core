package runtime

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestStateVMReads(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:10007",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xstatevm-genesis",
			StateRoot: "0xstatevm-state",
			BaseFee:   1,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	if err := rt.state.SetSenderNonce("alice", 3); err != nil {
		t.Fatalf("set nonce: %v", err)
	}
	if err := rt.state.SetBalance("alice", 1000); err != nil {
		t.Fatalf("set balance: %v", err)
	}
	if err := rt.state.SetCode("alice", []byte{0x60, 0x01, 0x60, 0x02}); err != nil {
		t.Fatalf("set code: %v", err)
	}
	if err := rt.state.SetStorage("alice", "0x01", "0xff"); err != nil {
		t.Fatalf("set storage: %v", err)
	}

	acct, err := rt.StateAccount("alice")
	if err != nil {
		t.Fatalf("state account: %v", err)
	}
	if !acct.Found {
		t.Fatalf("expected account found")
	}
	if acct.Nonce != 3 {
		t.Fatalf("unexpected nonce: got=%d want=3", acct.Nonce)
	}
	if acct.Balance != 1000 {
		t.Fatalf("unexpected balance: got=%d want=1000", acct.Balance)
	}
	if !acct.HasCode {
		t.Fatalf("expected hasCode=true")
	}
	if acct.StorageSlots != 1 {
		t.Fatalf("unexpected storage slots: got=%d want=1", acct.StorageSlots)
	}

	code, err := rt.StateCode("alice")
	if err != nil {
		t.Fatalf("state code: %v", err)
	}
	if !code.Found {
		t.Fatalf("expected code found")
	}
	if len(code.Code) != 4 {
		t.Fatalf("unexpected code len: got=%d want=4", len(code.Code))
	}

	storage, err := rt.StateStorage("alice", "0x01")
	if err != nil {
		t.Fatalf("state storage: %v", err)
	}
	if !storage.Found {
		t.Fatalf("expected storage found")
	}
	if storage.Value != "0xff" {
		t.Fatalf("unexpected storage value: got=%s want=0xff", storage.Value)
	}
}

func TestIntrospectionServer_ExposesStateVMEndpoints(t *testing.T) {
	rt, err := New(Config{
		ListenAddress: "127.0.0.1:10008",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "0xstatevm-http-genesis",
			StateRoot: "0xstatevm-http-state",
			BaseFee:   1,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	if err := rt.state.SetSenderNonce("alice", 4); err != nil {
		t.Fatalf("set nonce: %v", err)
	}
	if err := rt.state.SetBalance("alice", 2000); err != nil {
		t.Fatalf("set balance: %v", err)
	}
	if err := rt.state.SetCode("alice", []byte{0x60, 0x00}); err != nil {
		t.Fatalf("set code: %v", err)
	}
	if err := rt.state.SetStorage("alice", "0xaa", "0xbb"); err != nil {
		t.Fatalf("set storage: %v", err)
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

	var acct stateAccountResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &acct); err != nil {
		t.Fatalf("decode state account response: %v", err)
	}
	if !acct.Result.Found || acct.Result.Balance != 2000 || !acct.Result.HasCode {
		t.Fatalf("unexpected state account response")
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/state/code?address=alice", nil)
	server.Handler().ServeHTTP(rec2, req2)

	if rec2.Code != 200 {
		t.Fatalf("unexpected state code status code: got=%d want=200", rec2.Code)
	}

	var code stateCodeResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &code); err != nil {
		t.Fatalf("decode state code response: %v", err)
	}
	if !code.Result.Found || len(code.Result.Code) != 2 {
		t.Fatalf("unexpected state code response")
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/state/storage?address=alice&key=0xaa", nil)
	server.Handler().ServeHTTP(rec3, req3)

	if rec3.Code != 200 {
		t.Fatalf("unexpected state storage status code: got=%d want=200", rec3.Code)
	}

	var storage stateStorageResponse
	if err := json.Unmarshal(rec3.Body.Bytes(), &storage); err != nil {
		t.Fatalf("decode state storage response: %v", err)
	}
	if !storage.Result.Found || storage.Result.Value != "0xbb" {
		t.Fatalf("unexpected state storage response")
	}
}
