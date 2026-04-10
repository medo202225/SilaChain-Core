package consensus

import "testing"

func lifecycleTestDeposit(index uint64, data DepositData, siblings ...string) Deposit {
	dep := Deposit{
		Index: index,
		Proof: [][]string{siblings},
		Data:  data,
	}
	root, err := computeDepositRoot(dep)
	if err != nil {
		panic(err)
	}
	dep.Root = root
	return dep
}

func TestCanValidatorVoluntarilyExit(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(lifecycleTestDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	if state.CanValidatorVoluntarilyExit(0, state.Epoch) {
		t.Fatalf("validator should not be able to exit before activation")
	}

	state.ProcessRegistryUpdates()
	state.AdvanceSlot(96, 32)

	if !state.CanValidatorVoluntarilyExit(0, state.Epoch) {
		t.Fatalf("validator should be able to voluntarily exit after activation")
	}
}

func TestCanSlashValidator(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(lifecycleTestDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	if !state.CanSlashValidator(0) {
		t.Fatalf("validator should be slashable before slash")
	}

	state.SlashValidator(0)

	if state.CanSlashValidator(0) {
		t.Fatalf("validator should not be slashable after slash")
	}
}
