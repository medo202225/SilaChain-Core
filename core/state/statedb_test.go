package state

import "testing"

func TestStateDBSnapshotAndRevert(t *testing.T) {
	db := NewStateDB()

	db.SetBalance("alice", 10)
	snap := db.Snapshot()
	db.SetBalance("alice", 99)

	if got := db.GetBalance("alice"); got != 99 {
		t.Fatalf("balance before revert = %d", got)
	}

	db.RevertToSnapshot(snap)

	if got := db.GetBalance("alice"); got != 10 {
		t.Fatalf("balance after revert = %d", got)
	}
}

func TestStateDBIntermediateRootChangesWithState(t *testing.T) {
	db := NewStateDB()

	root1 := db.IntermediateRoot(false)

	db.SetBalance("alice", 10)
	db.SetNonce("alice", 1)
	db.SetState("alice", "k1", "v1")

	root2 := db.IntermediateRoot(false)

	if root1 == root2 {
		t.Fatalf("expected root to change")
	}
}

func TestStateDBCommitClearsJournalAndReturnsRoot(t *testing.T) {
	db := NewStateDB()

	db.SetBalance("alice", 10)
	db.SetState("alice", "k1", "v1")

	root, err := db.Commit(false)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}
	if root == "" {
		t.Fatalf("expected non-empty root")
	}
	if db.journal == nil {
		t.Fatalf("expected journal")
	}
	if len(db.journal.entries) != 0 {
		t.Fatalf("expected cleared journal entries")
	}
}

func TestStateDBRevertRefundAndCode(t *testing.T) {
	db := NewStateDB()

	snap := db.Snapshot()
	db.AddRefund(10)
	db.SetCode("alice", []byte{1, 2, 3})

	db.RevertToSnapshot(snap)

	if got := db.GetRefund(); got != 0 {
		t.Fatalf("refund = %d", got)
	}
	if got := db.GetCodeHash("alice"); got != "" {
		t.Fatalf("code hash = %q", got)
	}
}

func TestStateDBRevertTransientStorage(t *testing.T) {
	db := NewStateDB()

	snap := db.Snapshot()
	db.SetTransientState("alice", "k1", "v1")
	db.RevertToSnapshot(snap)

	if _, ok := db.GetTransientState("alice", "k1"); ok {
		t.Fatalf("expected transient storage revert")
	}
}
