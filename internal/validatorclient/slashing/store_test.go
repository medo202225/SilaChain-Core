package slashing

import (
	"context"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()

	path := filepath.Join(t.TempDir(), "slashing.sqlite")
	store := NewSQLiteStore(path)

	if err := store.Init(context.Background()); err != nil {
		t.Fatalf("init store: %v", err)
	}

	t.Cleanup(func() {
		_ = store.Close()
	})

	return store
}

func TestBlock_AllowsSameSlotSameRoot(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	pub := []byte("validator-a")
	root := []byte("root-1")

	if err := store.CheckAndRecordBlock(ctx, pub, 10, root); err != nil {
		t.Fatalf("first block record failed: %v", err)
	}

	if err := store.CheckAndRecordBlock(ctx, pub, 10, root); err != nil {
		t.Fatalf("same block record should be allowed: %v", err)
	}
}

func TestBlock_RejectsSameSlotDifferentRoot(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	pub := []byte("validator-a")

	if err := store.CheckAndRecordBlock(ctx, pub, 10, []byte("root-1")); err != nil {
		t.Fatalf("first block record failed: %v", err)
	}

	err := store.CheckAndRecordBlock(ctx, pub, 10, []byte("root-2"))
	if err != ErrSlashableBlock {
		t.Fatalf("expected ErrSlashableBlock, got %v", err)
	}
}

func TestAttestation_RejectsDoubleVote(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	pub := []byte("validator-a")

	if err := store.CheckAndRecordAttestation(ctx, pub, 1, 5, []byte("root-1")); err != nil {
		t.Fatalf("first attestation failed: %v", err)
	}

	err := store.CheckAndRecordAttestation(ctx, pub, 2, 5, []byte("root-2"))
	if err != ErrSlashableAttestation {
		t.Fatalf("expected ErrSlashableAttestation, got %v", err)
	}
}

func TestAttestation_RejectsSurroundVote(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	pub := []byte("validator-a")

	if err := store.CheckAndRecordAttestation(ctx, pub, 3, 6, []byte("root-1")); err != nil {
		t.Fatalf("first attestation failed: %v", err)
	}

	err := store.CheckAndRecordAttestation(ctx, pub, 2, 7, []byte("root-2"))
	if err != ErrSlashableAttestation {
		t.Fatalf("expected ErrSlashableAttestation, got %v", err)
	}
}

func TestAttestation_RejectsSurroundedVote(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	pub := []byte("validator-a")

	if err := store.CheckAndRecordAttestation(ctx, pub, 2, 7, []byte("root-1")); err != nil {
		t.Fatalf("first attestation failed: %v", err)
	}

	err := store.CheckAndRecordAttestation(ctx, pub, 3, 6, []byte("root-2"))
	if err != ErrSlashableAttestation {
		t.Fatalf("expected ErrSlashableAttestation, got %v", err)
	}
}
