package execution

import "testing"

func TestStateImportAndPendingTx(t *testing.T) {
	s := NewState("0xgenesis")

	if s.HeadNumber() != 0 {
		t.Fatalf("unexpected genesis head number: %d", s.HeadNumber())
	}
	if s.HeadHash() != "0xgenesis" {
		t.Fatalf("unexpected genesis head hash: %s", s.HeadHash())
	}

	ok := s.AddPendingTx(PendingTx{
		Hash:  "0xtx1",
		From:  "a",
		To:    "b",
		Value: "10",
		Nonce: 1,
	})
	if !ok {
		t.Fatal("expected pending tx insertion")
	}
	if !s.HasPendingTx("0xtx1") {
		t.Fatal("expected tx in pending pool")
	}
	if s.PendingCount() != 1 {
		t.Fatalf("unexpected pending count: %d", s.PendingCount())
	}

	err := s.ImportBlock(ImportedBlock{
		Number:     1,
		Hash:       "0xblock1",
		ParentHash: "0xgenesis",
		Timestamp:  1,
		TxHashes:   []string{"0xtx1"},
	})
	if err != nil {
		t.Fatalf("ImportBlock failed: %v", err)
	}

	if s.HeadNumber() != 1 {
		t.Fatalf("unexpected head number after import: %d", s.HeadNumber())
	}
	if s.HeadHash() != "0xblock1" {
		t.Fatalf("unexpected head hash after import: %s", s.HeadHash())
	}
}
