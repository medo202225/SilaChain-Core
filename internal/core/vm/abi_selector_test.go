package vm

import "testing"

func TestFunctionSelectorTransfer(t *testing.T) {
	got := FunctionSelectorHex("transfer(address,uint256)")
	want := "a9059cbb"

	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestFunctionSelectorBalanceOf(t *testing.T) {
	got := FunctionSelectorHex("balanceOf(address)")
	want := "70a08231"

	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestMatchFunctionSelector(t *testing.T) {
	input := []byte{0xa9, 0x05, 0x9c, 0xbb, 0x00, 0x01}
	if !MatchFunctionSelector(input, "transfer(address,uint256)") {
		t.Fatalf("expected selector match")
	}
}

func TestNormalizeFunctionSignature(t *testing.T) {
	got := NormalizeFunctionSignature("  balanceOf(address)  ")
	if got != "balanceOf(address)" {
		t.Fatalf("expected trimmed signature, got %q", got)
	}
}
