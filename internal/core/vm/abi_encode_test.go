package vm

import "testing"

func TestEncodeABICallTransfer(t *testing.T) {
	got, err := EncodeABICallHex(
		"transfer(address,uint256)",
		ABIArgument{Type: ABITypeAddress, Value: "0x1111111111111111111111111111111111111111"},
		ABIArgument{Type: ABITypeUint256, Value: uint64(5)},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 8+(64*2) {
		t.Fatalf("unexpected encoded length: %d", len(got))
	}
	if got[:8] != "a9059cbb" {
		t.Fatalf("expected selector a9059cbb, got %s", got[:8])
	}
}

func TestEncodeABIArgumentBool(t *testing.T) {
	encoded, err := EncodeABIArgument(ABIArgument{Type: ABITypeBool, Value: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(encoded) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(encoded))
	}
	if encoded[31] != 1 {
		t.Fatalf("expected bool true encoding")
	}
}

func TestEncodeABIArgumentUint256(t *testing.T) {
	encoded, err := EncodeABIArgument(ABIArgument{Type: ABITypeUint256, Value: uint64(255)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(encoded) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(encoded))
	}
	if encoded[31] != 0xff {
		t.Fatalf("expected last byte ff")
	}
}

func TestDecodeABIReturnBool(t *testing.T) {
	data := make([]byte, 32)
	data[31] = 1

	got, err := DecodeABIReturnBool(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatalf("expected true")
	}
}

func TestDecodeABIReturnUint256(t *testing.T) {
	data := make([]byte, 32)
	data[31] = 7

	got, err := DecodeABIReturnUint256(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "7" {
		t.Fatalf("expected 7, got %s", got)
	}
}
