package sdk

import "testing"

func TestBuildDeployTx(t *testing.T) {
	req := BuildDeployTx("alice", "6001600055", 1, 1)

	if req.From != "alice" {
		t.Fatalf("expected from alice")
	}
	if req.VMVersion != 1 {
		t.Fatalf("expected vm version 1")
	}
	if req.ContractCode == "" {
		t.Fatalf("expected contract code")
	}
}

func TestBuildCallTx(t *testing.T) {
	req := BuildCallTx("alice", "contract1", "a9059cbb", 2, 1)

	if req.Address != "contract1" {
		t.Fatalf("expected contract1")
	}
	if req.ContractInput != "a9059cbb" {
		t.Fatalf("expected calldata")
	}
}

func TestSignDeployTxMissingKey(t *testing.T) {
	_, err := SignDeployTx(BuildDeployTx("alice", "6001600055", 1, 1), "")
	if err == nil {
		t.Fatalf("expected missing key error")
	}
}

func TestSignCallTxMissingKey(t *testing.T) {
	_, err := SignCallTx(BuildCallTx("alice", "contract1", "a9059cbb", 1, 1), "")
	if err == nil {
		t.Fatalf("expected missing key error")
	}
}
