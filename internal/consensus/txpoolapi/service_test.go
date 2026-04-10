package txpoolapi

import (
	"testing"

	"silachain/internal/consensus/txpool"
)

type testState struct {
	nonces map[string]uint64
}

func newTestState() *testState {
	return &testState{
		nonces: make(map[string]uint64),
	}
}

func (s *testState) SenderNonce(sender string) uint64 {
	return s.nonces[sender]
}

func TestAdd_IngestsTransactionIntoRealTxPool(t *testing.T) {
	pool := txpool.NewPool(10)
	state := newTestState()

	svc, err := New(pool, state)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := svc.Add(AddTxRequest{
		Hash:                 "tx-1",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            1,
	})
	if err != nil {
		t.Fatalf("add tx: %v", err)
	}

	if !result.Accepted {
		t.Fatalf("expected accepted=true")
	}
	if result.PendingCount != 1 {
		t.Fatalf("unexpected pending count: got=%d want=1", result.PendingCount)
	}
	if pool.PendingCount() != 1 {
		t.Fatalf("unexpected pool pending count: got=%d want=1", pool.PendingCount())
	}
}

func TestStatus_ReturnsRealPoolState(t *testing.T) {
	pool := txpool.NewPool(15)
	state := newTestState()

	svc, err := New(pool, state)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	status, err := svc.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}

	if status.PendingCount != 0 {
		t.Fatalf("unexpected pending count: got=%d want=0", status.PendingCount)
	}
	if status.BaseFee != 15 {
		t.Fatalf("unexpected base fee: got=%d want=15", status.BaseFee)
	}
}
