package sdk

import "testing"

func TestEncodeCallDataTransfer(t *testing.T) {
	got, err := EncodeCallData(
		"transfer(address,uint256)",
		ABIArgument{Type: ABITypeAddress, Value: "0x1111111111111111111111111111111111111111"},
		ABIArgument{Type: ABITypeUint256, Value: uint64(5)},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected calldata")
	}
	if got[:8] != "a9059cbb" {
		t.Fatalf("expected selector a9059cbb, got %s", got[:8])
	}
}

func TestNewReadOnlyCall(t *testing.T) {
	req := NewReadOnlyCall(ReadOnlyCallRequest{
		To:    "SILA_CONTRACT_001",
		Input: "a9059cbb",
	})

	if req.Method != "sila_call" {
		t.Fatalf("expected sila_call, got %s", req.Method)
	}
	if len(req.Params) != 1 {
		t.Fatalf("expected one param object")
	}
}
