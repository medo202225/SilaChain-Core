package consensus

import "testing"

func exitTestDeposit(index uint64, data DepositData, siblings ...string) Deposit {
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

func TestInitiateValidatorExitSetsExitAndWithdrawableEpoch(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(exitTestDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	state.ProcessRegistryUpdates()
	state.AdvanceSlot(96, 32)

	state.InitiateValidatorExit(0)

	if state.Validators[0].ExitEpoch != state.Epoch+1 {
		t.Fatalf("expected exit epoch %d, got %d", state.Epoch+1, state.Validators[0].ExitEpoch)
	}
	if state.Validators[0].WithdrawableEpoch != state.Validators[0].ExitEpoch+minWithdrawabilityDelay {
		t.Fatalf("expected withdrawable epoch %d, got %d", state.Validators[0].ExitEpoch+minWithdrawabilityDelay, state.Validators[0].WithdrawableEpoch)
	}
}

func TestValidatorStopsBeingActiveAtExitEpoch(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(exitTestDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	state.ProcessRegistryUpdates()
	state.AdvanceSlot(96, 32)

	if len(state.ActiveValidators()) != 1 {
		t.Fatalf("expected validator to be active before exit")
	}

	state.InitiateValidatorExit(0)
	exitEpoch := state.Validators[0].ExitEpoch

	state.AdvanceSlot(exitEpoch*32, 32)

	if len(state.ActiveValidators()) != 0 {
		t.Fatalf("expected validator to stop being active at exit epoch")
	}
}

func TestSlashValidatorSetsSlashedExitAndPenalty(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(exitTestDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	beforeBalance := state.Balances[0]
	state.SlashValidator(0)

	if !state.Validators[0].Slashed {
		t.Fatalf("expected validator to be slashed")
	}
	if state.Validators[0].ExitEpoch == farFutureEpoch {
		t.Fatalf("expected slashed validator to have exit epoch")
	}
	if state.Balances[0] >= beforeBalance {
		t.Fatalf("expected balance penalty after slashing")
	}
	if state.Validators[0].WithdrawableEpoch == farFutureEpoch {
		t.Fatalf("expected withdrawable epoch to be set after slashing")
	}
}

func TestSlashedValidatorBalanceZeroedAtWithdrawableEpoch(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(exitTestDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	state.SlashValidator(0)

	if state.Balances[0] == 0 {
		t.Fatalf("expected balance to remain non-zero immediately after slashing")
	}

	withdrawableEpoch := state.Validators[0].WithdrawableEpoch
	state.AdvanceSlot(withdrawableEpoch*32, 32)

	if state.Balances[0] != 0 {
		t.Fatalf("expected slashed validator balance to be zeroed at withdrawable epoch, got %d", state.Balances[0])
	}
	if state.Validators[0].EffectiveBalance != 0 {
		t.Fatalf("expected effective balance to be zeroed at withdrawable epoch, got %d", state.Validators[0].EffectiveBalance)
	}
}
