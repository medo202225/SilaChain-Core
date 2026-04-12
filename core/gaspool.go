package core

import "fmt"

type GasPool struct {
	gas uint64
}

func NewGasPool(gas uint64) *GasPool {
	return new(GasPool).AddGas(gas)
}

func (gp *GasPool) AddGas(amount uint64) *GasPool {
	if gp == nil {
		return nil
	}
	gp.gas += amount
	return gp
}

func (gp *GasPool) Gas() uint64 {
	if gp == nil {
		return 0
	}
	return gp.gas
}

func (gp *GasPool) SubGas(amount uint64) error {
	if gp == nil {
		return fmt.Errorf("gas pool is nil")
	}
	if gp.gas < amount {
		return fmt.Errorf("gas pool underflow")
	}
	gp.gas -= amount
	return nil
}

func (gp *GasPool) ReturnGas(amount uint64) error {
	if gp == nil {
		return fmt.Errorf("gas pool is nil")
	}
	gp.gas += amount
	return nil
}
