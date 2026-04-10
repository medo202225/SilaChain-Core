package consensus

import "testing"

func classificationDeposit(index uint64, data DepositData, siblings ...string) Deposit {
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

func TestLifecycleClassificationExitedAndWithdrawable(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(classificationDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	state.ProcessRegistryUpdates()
	state.AdvanceSlot(96, 32)

	state.InitiateValidatorExit(0)

	exitEpoch := state.Validators[0].ExitEpoch
	state.AdvanceSlot(exitEpoch*32, 32)

	if len(state.ExitedValidators()) != 1 {
		t.Fatalf("expected 1 exited validator, got %d", len(state.ExitedValidators()))
	}
	if len(state.ExitedButNotWithdrawableValidators()) != 1 {
		t.Fatalf("expected 1 exited but not withdrawable validator, got %d", len(state.ExitedButNotWithdrawableValidators()))
	}
	if len(state.WithdrawableValidators()) != 0 {
		t.Fatalf("expected 0 withdrawable validators before withdrawable epoch")
	}

	withdrawableEpoch := state.Validators[0].WithdrawableEpoch
	state.AdvanceSlot(withdrawableEpoch*32, 32)

	if len(state.WithdrawableValidators()) != 1 {
		t.Fatalf("expected 1 withdrawable validator, got %d", len(state.WithdrawableValidators()))
	}
}

func TestLifecycleClassificationSlashedWithdrawable(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(classificationDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	state.SlashValidator(0)

	if len(state.SlashedValidators()) != 1 {
		t.Fatalf("expected 1 slashed validator, got %d", len(state.SlashedValidators()))
	}

	withdrawableEpoch := state.Validators[0].WithdrawableEpoch
	state.AdvanceSlot(withdrawableEpoch*32, 32)

	if len(state.WithdrawableValidators()) != 1 {
		t.Fatalf("expected slashed validator to become withdrawable")
	}
}
