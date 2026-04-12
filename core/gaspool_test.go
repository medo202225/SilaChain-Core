package core

import "testing"

func TestGasPoolSubGas(t *testing.T) {
	gp := NewGasPool(100)

	if err := gp.SubGas(40); err != nil {
		t.Fatalf("sub gas: %v", err)
	}
	if got := gp.Gas(); got != 60 {
		t.Fatalf("gas = %d", got)
	}
}

func TestGasPoolUnderflow(t *testing.T) {
	gp := NewGasPool(10)

	if err := gp.SubGas(11); err == nil {
		t.Fatalf("expected underflow error")
	}
}

func TestGasPoolReturnGas(t *testing.T) {
	gp := NewGasPool(10)

	if err := gp.SubGas(4); err != nil {
		t.Fatalf("sub gas: %v", err)
	}
	if err := gp.ReturnGas(4); err != nil {
		t.Fatalf("return gas: %v", err)
	}
	if got := gp.Gas(); got != 10 {
		t.Fatalf("gas = %d", got)
	}
}
