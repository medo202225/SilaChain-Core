package consensus

import "testing"

func testDepositRoot(index uint64, data DepositData, siblings ...string) string {
	dep := Deposit{
		Index: index,
		Proof: [][]string{siblings},
		Data:  data,
	}
	root, err := computeDepositRoot(dep)
	if err != nil {
		panic(err)
	}
	return root
}

func mustDeposit(index uint64, data DepositData, siblings ...string) Deposit {
	return Deposit{
		Index: index,
		Root:  testDepositRoot(index, data, siblings...),
		Proof: [][]string{siblings},
		Data:  data,
	}
}

func TestProcessDepositAddsNewValidator(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	dep := mustDeposit(0, data, "0xabc123")

	index, created, err := state.ProcessDeposit(dep)
	if err != nil {
		t.Fatalf("process deposit failed: %v", err)
	}
	if !created {
		t.Fatalf("expected new validator to be created")
	}
	if index != 0 {
		t.Fatalf("expected validator index 0, got %d", index)
	}
	if len(state.Validators) != 1 {
		t.Fatalf("expected 1 validator, got %d", len(state.Validators))
	}
	if len(state.Balances) != 1 {
		t.Fatalf("expected 1 balance, got %d", len(state.Balances))
	}
	if state.Balances[0] != 32000000000 {
		t.Fatalf("expected balance 32000000000, got %d", state.Balances[0])
	}
	if state.Validators[0].EffectiveBalance != 32000000000 {
		t.Fatalf("expected effective balance 32000000000, got %d", state.Validators[0].EffectiveBalance)
	}
	if state.Validators[0].ActivationEligibilityEpoch != state.Epoch {
		t.Fatalf("expected activation eligibility epoch %d, got %d", state.Epoch, state.Validators[0].ActivationEligibilityEpoch)
	}
	if state.Validators[0].ActivationEpoch != farFutureEpoch {
		t.Fatalf("expected activation epoch farFutureEpoch, got %d", state.Validators[0].ActivationEpoch)
	}
	if state.Eth1DepositIndex != 1 {
		t.Fatalf("expected eth1 deposit index 1, got %d", state.Eth1DepositIndex)
	}
	if state.DepositRoot != dep.Root {
		t.Fatalf("expected deposit root to match processed deposit root")
	}
}

func TestProcessDepositTopsUpExistingValidator(t *testing.T) {
	state := NewBeaconStateV1(nil)

	data1 := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                16000000000,
		Signature:             "sig-1",
	}
	dep1 := mustDeposit(0, data1, "0xaaa111")

	_, created, err := state.ProcessDeposit(dep1)
	if err != nil {
		t.Fatalf("first deposit failed: %v", err)
	}
	if !created {
		t.Fatalf("expected first deposit to create validator")
	}

	data2 := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                16000000000,
		Signature:             "sig-2",
	}
	dep2 := mustDeposit(1, data2, "0xbbb222")

	_, created, err = state.ProcessDeposit(dep2)
	if err != nil {
		t.Fatalf("second deposit failed: %v", err)
	}
	if created {
		t.Fatalf("expected second deposit to top up existing validator")
	}
	if len(state.Validators) != 1 {
		t.Fatalf("expected 1 validator, got %d", len(state.Validators))
	}
	if state.Balances[0] != 32000000000 {
		t.Fatalf("expected total balance 32000000000, got %d", state.Balances[0])
	}
	if state.Validators[0].EffectiveBalance != 32000000000 {
		t.Fatalf("expected effective balance 32000000000, got %d", state.Validators[0].EffectiveBalance)
	}
	if state.Eth1DepositIndex != 2 {
		t.Fatalf("expected eth1 deposit index 2, got %d", state.Eth1DepositIndex)
	}
	if state.DepositRoot != dep2.Root {
		t.Fatalf("expected deposit root to match second deposit root")
	}
}

func TestProcessDepositRejectsUnexpectedDepositIndex(t *testing.T) {
	state := NewBeaconStateV1(nil)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	dep := mustDeposit(1, data, "0xabc123")

	_, _, err := state.ProcessDeposit(dep)
	if err == nil {
		t.Fatalf("expected deposit with wrong index to fail")
	}
}

func TestProcessDepositRejectsMissingProof(t *testing.T) {
	state := NewBeaconStateV1(nil)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	dep := Deposit{
		Index: 0,
		Root:  "0xdeadbeef",
		Proof: nil,
		Data:  data,
	}

	_, _, err := state.ProcessDeposit(dep)
	if err == nil {
		t.Fatalf("expected deposit with missing proof to fail")
	}
}

func TestProcessDepositRejectsInvalidProofRoot(t *testing.T) {
	state := NewBeaconStateV1(nil)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	dep := Deposit{
		Index: 0,
		Root:  "0xdeadbeef",
		Proof: [][]string{{"0xabc123"}},
		Data:  data,
	}

	_, _, err := state.ProcessDeposit(dep)
	if err == nil {
		t.Fatalf("expected deposit with invalid proof root to fail")
	}
}
