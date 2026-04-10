package txpool

import "testing"

func TestOrdered_RespectsSenderNonceChainAndCrossSenderPriority(t *testing.T) {
	pool := NewPool(10)

	if err := pool.SetSenderStateNonce("alice", 0); err != nil {
		t.Fatalf("set alice state nonce: %v", err)
	}
	if err := pool.SetSenderStateNonce("bob", 0); err != nil {
		t.Fatalf("set bob state nonce: %v", err)
	}

	errs := []error{
		pool.Add(Tx{Hash: "alice-1", From: "alice", Nonce: 1, GasLimit: 21000, MaxFeePerGas: 500, MaxPriorityFeePerGas: 100, Timestamp: 2}),
		pool.Add(Tx{Hash: "alice-0", From: "alice", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 20, MaxPriorityFeePerGas: 2, Timestamp: 1}),
		pool.Add(Tx{Hash: "bob-0", From: "bob", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 100, MaxPriorityFeePerGas: 50, Timestamp: 1}),
	}
	for _, err := range errs {
		if err != nil {
			t.Fatalf("add tx: %v", err)
		}
	}

	ordered := pool.Ordered()
	if len(ordered) != 3 {
		t.Fatalf("unexpected tx count: got=%d want=3", len(ordered))
	}

	got := []string{ordered[0].Hash, ordered[1].Hash, ordered[2].Hash}
	want := []string{"bob-0", "alice-0", "alice-1"}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected order at index %d: got=%s want=%s full=%v", i, got[i], want[i], got)
		}
	}
}

func TestAdd_ReplacementRequiresHigherEffectiveFee(t *testing.T) {
	pool := NewPool(10)

	if err := pool.SetSenderStateNonce("alice", 0); err != nil {
		t.Fatalf("set sender state nonce: %v", err)
	}

	if err := pool.Add(Tx{
		Hash:                 "old",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            1,
	}); err != nil {
		t.Fatalf("add old tx: %v", err)
	}

	if err := pool.Add(Tx{
		Hash:                 "low",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            2,
	}); err == nil {
		t.Fatalf("expected underpriced replacement error")
	}

	if err := pool.Add(Tx{
		Hash:                 "new",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         50,
		MaxPriorityFeePerGas: 10,
		Timestamp:            3,
	}); err != nil {
		t.Fatalf("add replacement tx: %v", err)
	}

	ordered := pool.Ordered()
	if len(ordered) != 1 {
		t.Fatalf("unexpected ordered count: got=%d want=1", len(ordered))
	}

	if ordered[0].Hash != "new" {
		t.Fatalf("unexpected replacement winner: got=%s want=new", ordered[0].Hash)
	}
}

func TestRemoveIncluded_PrunesCanonicalTransactionsAndAdvancesStateNonce(t *testing.T) {
	pool := NewPool(1)

	if err := pool.SetSenderStateNonce("alice", 0); err != nil {
		t.Fatalf("set alice nonce: %v", err)
	}
	if err := pool.SetSenderStateNonce("bob", 0); err != nil {
		t.Fatalf("set bob nonce: %v", err)
	}

	for _, tx := range []Tx{
		{Hash: "alice-0", From: "alice", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 20, MaxPriorityFeePerGas: 2, Timestamp: 1},
		{Hash: "alice-1", From: "alice", Nonce: 1, GasLimit: 21000, MaxFeePerGas: 21, MaxPriorityFeePerGas: 2, Timestamp: 2},
		{Hash: "bob-0", From: "bob", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 30, MaxPriorityFeePerGas: 3, Timestamp: 1},
	} {
		if err := pool.Add(tx); err != nil {
			t.Fatalf("add tx %s: %v", tx.Hash, err)
		}
	}

	if err := pool.RemoveIncluded([]Tx{
		{Hash: "bob-0", From: "bob", Nonce: 0},
		{Hash: "alice-0", From: "alice", Nonce: 0},
	}); err != nil {
		t.Fatalf("remove included: %v", err)
	}

	if pool.PendingCount() != 1 {
		t.Fatalf("unexpected pending count after prune: got=%d want=1", pool.PendingCount())
	}
	if pool.SenderStateNonce("alice") != 1 {
		t.Fatalf("unexpected alice state nonce: got=%d want=1", pool.SenderStateNonce("alice"))
	}
	if pool.SenderStateNonce("bob") != 1 {
		t.Fatalf("unexpected bob state nonce: got=%d want=1", pool.SenderStateNonce("bob"))
	}

	ordered := pool.Ordered()
	if len(ordered) != 1 {
		t.Fatalf("unexpected ordered count after prune: got=%d want=1", len(ordered))
	}
	if ordered[0].Hash != "alice-1" {
		t.Fatalf("unexpected remaining tx: got=%s want=alice-1", ordered[0].Hash)
	}
}
