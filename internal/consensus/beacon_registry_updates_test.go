package consensus

import "testing"

func registryTestDeposit(index uint64, data DepositData, siblings ...string) Deposit {
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

func TestProcessRegistryUpdatesActivatesOnlyUpToChurnLimit(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	for i := 0; i < 2; i++ {
		data := DepositData{
			PublicKey:             "validator-" + string(rune('1'+i)),
			WithdrawalCredentials: "withdrawal",
			Amount:                32000000000,
			Signature:             "sig",
		}
		_, _, err := state.ProcessDeposit(registryTestDeposit(uint64(i), data, "0xabc123"))
		if err != nil {
			t.Fatalf("deposit failed: %v", err)
		}
	}

	state.ProcessRegistryUpdates()

	activatedCount := 0
	for _, v := range state.Validators {
		if v.ActivationEpoch != farFutureEpoch {
			activatedCount++
		}
	}

	if activatedCount != 1 {
		t.Fatalf("expected exactly 1 validator activated due to churn limit, got %d", activatedCount)
	}
}

func TestActivatedValidatorBecomesActiveOnNextEpoch(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(registryTestDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	state.ProcessRegistryUpdates()

	if len(state.ActiveValidators()) != 0 {
		t.Fatalf("expected no active validators before activation epoch")
	}

	state.AdvanceSlot(96, 32)

	if len(state.ActiveValidators()) != 1 {
		t.Fatalf("expected validator to become active in next epoch, got %d", len(state.ActiveValidators()))
	}
}
