package consensus

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBeaconStateFromDepositSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "validators.json")

	raw := `[
  {
    "public_key": "validator-1",
    "withdrawal_credentials": "withdrawal-1",
    "effective_balance": 32000000000,
    "slashed": false,
    "activation_epoch": 0,
    "exit_epoch": 18446744073709551615,
    "withdrawable_epoch": 18446744073709551615
  },
  {
    "public_key": "validator-2",
    "withdrawal_credentials": "withdrawal-2",
    "effective_balance": 32000000000,
    "slashed": false,
    "activation_epoch": 0,
    "exit_epoch": 18446744073709551615,
    "withdrawable_epoch": 18446744073709551615
  }
]`

	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write validators file failed: %v", err)
	}

	state, err := LoadBeaconStateFromDepositSource(path, 32)
	if err != nil {
		t.Fatalf("load beacon state from deposit source failed: %v", err)
	}

	if len(state.Validators) != 2 {
		t.Fatalf("expected 2 validators, got %d", len(state.Validators))
	}
	if len(state.Balances) != 2 {
		t.Fatalf("expected 2 balances, got %d", len(state.Balances))
	}
	if state.Eth1DepositIndex != 2 {
		t.Fatalf("expected eth1 deposit index 2, got %d", state.Eth1DepositIndex)
	}
	if state.DepositRoot == "" {
		t.Fatalf("expected deposit root to be set")
	}
	if state.Eth1Data.DepositRoot != state.DepositRoot {
		t.Fatalf("expected eth1 deposit root to match state deposit root")
	}
	if state.Eth1Data.DepositCount != state.Eth1DepositIndex {
		t.Fatalf("expected eth1 deposit count %d, got %d", state.Eth1DepositIndex, state.Eth1Data.DepositCount)
	}
}
