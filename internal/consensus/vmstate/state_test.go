package vmstate

import "testing"

func TestState_AccountLifecycle(t *testing.T) {
	st := New()

	if err := st.SetNonce("alice", 7); err != nil {
		t.Fatalf("set nonce: %v", err)
	}
	if err := st.SetBalance("alice", 100); err != nil {
		t.Fatalf("set balance: %v", err)
	}
	if err := st.SetCode("alice", []byte{0x60, 0x00}); err != nil {
		t.Fatalf("set code: %v", err)
	}
	if err := st.SetStorage("alice", "0x01", "0xab"); err != nil {
		t.Fatalf("set storage: %v", err)
	}

	acct, ok := st.GetAccount("alice")
	if !ok {
		t.Fatalf("expected account to exist")
	}
	if acct.Nonce != 7 {
		t.Fatalf("unexpected nonce: got=%d want=7", acct.Nonce)
	}
	if acct.Balance != 100 {
		t.Fatalf("unexpected balance: got=%d want=100", acct.Balance)
	}
	if !acct.HasCode() {
		t.Fatalf("expected hasCode=true")
	}
	if acct.StorageSlots() != 1 {
		t.Fatalf("unexpected storage slots: got=%d want=1", acct.StorageSlots())
	}

	code, ok := st.GetCode("alice")
	if !ok {
		t.Fatalf("expected code to exist")
	}
	if len(code) != 2 {
		t.Fatalf("unexpected code len: got=%d want=2", len(code))
	}

	val, ok := st.GetStorage("alice", "0x01")
	if !ok {
		t.Fatalf("expected storage value")
	}
	if val != "0xab" {
		t.Fatalf("unexpected storage value: got=%s want=0xab", val)
	}
}
