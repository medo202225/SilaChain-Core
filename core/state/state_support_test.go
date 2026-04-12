package state

import "testing"

func TestAccessListAddressAndSlot(t *testing.T) {
	al := NewAccessList()
	al.AddAddress("alice")
	al.AddSlot("alice", "slot1")

	if !al.ContainsAddress("alice") {
		t.Fatalf("expected alice in access list")
	}
	addrOK, slotOK := al.Contains("alice", "slot1")
	if !addrOK || !slotOK {
		t.Fatalf("expected alice slot1 in access list")
	}
}

func TestTransientStorageSetGetReset(t *testing.T) {
	ts := NewTransientStorage()
	ts.Set("alice", "k1", "v1")

	if got, ok := ts.Get("alice", "k1"); !ok || got != "v1" {
		t.Fatalf("unexpected transient value: %q %v", got, ok)
	}

	ts.Reset()
	if _, ok := ts.Get("alice", "k1"); ok {
		t.Fatalf("expected reset transient storage")
	}
}

func TestStateDBRefundAndLogs(t *testing.T) {
	db := NewStateDB()

	db.AddRefund(10)
	db.AddLog(StateLog{
		Address: "alice",
		Topics:  []string{"t1"},
		Data:    []byte("hello"),
	})

	if got := db.GetRefund(); got != 10 {
		t.Fatalf("refund = %d", got)
	}
	if got := len(db.Logs()); got != 1 {
		t.Fatalf("logs = %d", got)
	}
}

func TestStateDBPrepareResetsTransientStorageAndLoadsAccessList(t *testing.T) {
	db := NewStateDB()
	db.SetTransientState("alice", "temp", "x")

	to := "bob"
	db.Prepare(nil, "alice", "coinbase", &to, []string{"pre1"}, []AccessTuple{
		{
			Address:     "charlie",
			StorageKeys: []string{"slot1"},
		},
	})

	if _, ok := db.GetTransientState("alice", "temp"); ok {
		t.Fatalf("expected transient storage reset")
	}
	if !db.AddressInAccessList("alice") {
		t.Fatalf("expected sender in access list")
	}
	if !db.AddressInAccessList("coinbase") {
		t.Fatalf("expected coinbase in access list")
	}
	if !db.AddressInAccessList("bob") {
		t.Fatalf("expected recipient in access list")
	}
	if !db.AddressInAccessList("pre1") {
		t.Fatalf("expected precompile in access list")
	}
	addrOK, slotOK := db.SlotInAccessList("charlie", "slot1")
	if !addrOK || !slotOK {
		t.Fatalf("expected charlie slot1 in access list")
	}
}

func TestStateDBObjectCodeAndTouch(t *testing.T) {
	db := NewStateDB()
	db.SetCode("alice", []byte{1, 2, 3})
	db.Touch("alice")

	if got := db.GetCodeHash("alice"); got == "" {
		t.Fatalf("expected code hash")
	}
	if !db.Exist("alice") {
		t.Fatalf("expected alice to exist")
	}
	if db.Empty("alice") {
		t.Fatalf("expected alice not empty after code")
	}
}

func TestStateDBObjectBalanceOps(t *testing.T) {
	db := NewStateDB()
	db.AddBalance("alice", 15)
	db.SubBalance("alice", 5)

	if got := db.GetBalance("alice"); got != 10 {
		t.Fatalf("balance = %d", got)
	}
}
