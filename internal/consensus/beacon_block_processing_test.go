package consensus

import "testing"

func blockTestDeposit(index uint64, data DepositData, siblings ...string) Deposit {
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

func TestProcessBlockProcessesDeposits(t *testing.T) {
	state := NewBeaconStateV1(nil)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig-1",
	}

	dep := blockTestDeposit(0, data, "0xabc123")

	block := BeaconBlock{
		Slot: 64,
		Body: BeaconBlockBody{
			Eth1Data: Eth1Data{
				DepositRoot:  dep.Root,
				DepositCount: 1,
				BlockHash:    "0xeth1block1",
			},
			Deposits: []Deposit{dep},
		},
	}

	if err := state.ProcessBlock(block, 32); err != nil {
		t.Fatalf("process block failed: %v", err)
	}

	if state.Slot != 64 {
		t.Fatalf("expected slot 64, got %d", state.Slot)
	}
	if state.Epoch != 2 {
		t.Fatalf("expected epoch 2, got %d", state.Epoch)
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
	if state.Validators[0].ActivationEligibilityEpoch != state.Epoch {
		t.Fatalf("expected activation eligibility epoch %d, got %d", state.Epoch, state.Validators[0].ActivationEligibilityEpoch)
	}
	if state.Eth1DepositIndex != 1 {
		t.Fatalf("expected eth1 deposit index 1, got %d", state.Eth1DepositIndex)
	}
	if state.Eth1Data.DepositRoot != dep.Root {
		t.Fatalf("expected eth1 deposit root to match deposit root")
	}
}

func TestProcessBlockRejectsMismatchedEth1DepositCount(t *testing.T) {
	state := NewBeaconStateV1(nil)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig-1",
	}
	dep := blockTestDeposit(0, data, "0xabc123")

	block := BeaconBlock{
		Slot: 64,
		Body: BeaconBlockBody{
			Eth1Data: Eth1Data{
				DepositRoot:  dep.Root,
				DepositCount: 2,
				BlockHash:    "0xeth1block1",
			},
			Deposits: []Deposit{dep},
		},
	}

	if err := state.ProcessBlock(block, 32); err == nil {
		t.Fatalf("expected eth1 deposit count mismatch to fail")
	}
}

func TestProcessBlockProcessesVoluntaryExit(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(blockTestDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	state.ProcessRegistryUpdates()
	state.AdvanceSlot(96, 32)

	block := BeaconBlock{
		Slot: 97,
		Body: BeaconBlockBody{
			Eth1Data: Eth1Data{
				DepositRoot:  state.DepositRoot,
				DepositCount: state.Eth1DepositIndex,
				BlockHash:    "0xeth1block2",
			},
			VoluntaryExits: []VoluntaryExit{
				{
					ValidatorIndex: 0,
					Epoch:          state.Epoch,
					Signature:      "exit-sig",
				},
			},
		},
	}

	if err := state.ProcessBlock(block, 32); err != nil {
		t.Fatalf("process block with voluntary exit failed: %v", err)
	}

	if state.Validators[0].ExitEpoch == farFutureEpoch {
		t.Fatalf("expected exit epoch to be set")
	}
	if state.Validators[0].WithdrawableEpoch == farFutureEpoch {
		t.Fatalf("expected withdrawable epoch to be set")
	}
}

func TestProcessBlockProcessesSlashing(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(blockTestDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	beforeBalance := state.Balances[0]

	block := BeaconBlock{
		Slot: 65,
		Body: BeaconBlockBody{
			Eth1Data: Eth1Data{
				DepositRoot:  state.DepositRoot,
				DepositCount: state.Eth1DepositIndex,
				BlockHash:    "0xeth1block3",
			},
			Slashings: []SlashingOperation{
				{
					ValidatorIndex: 0,
					Reason:         "double_vote",
				},
			},
		},
	}

	if err := state.ProcessBlock(block, 32); err != nil {
		t.Fatalf("process block with slashing failed: %v", err)
	}

	if !state.Validators[0].Slashed {
		t.Fatalf("expected validator to be slashed")
	}
	if state.Balances[0] >= beforeBalance {
		t.Fatalf("expected slashing penalty to reduce balance")
	}
}

func TestProcessBlockRejectsOldSlot(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	block := BeaconBlock{
		Slot: 63,
		Body: BeaconBlockBody{},
	}

	err := state.ProcessBlock(block, 32)
	if err == nil {
		t.Fatalf("expected old slot block to fail")
	}
}

func TestProcessBlockRejectsFutureVoluntaryExitEpoch(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(blockTestDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	state.ProcessRegistryUpdates()
	state.AdvanceSlot(96, 32)

	block := BeaconBlock{
		Slot: 97,
		Body: BeaconBlockBody{
			Eth1Data: Eth1Data{
				DepositRoot:  state.DepositRoot,
				DepositCount: state.Eth1DepositIndex,
				BlockHash:    "0xeth1block4",
			},
			VoluntaryExits: []VoluntaryExit{
				{
					ValidatorIndex: 0,
					Epoch:          state.Epoch + 1,
					Signature:      "bad-exit-sig",
				},
			},
		},
	}

	if err := state.ProcessBlock(block, 32); err == nil {
		t.Fatalf("expected future voluntary exit epoch to fail")
	}
}

func TestProcessBlockRejectsDuplicateSlashing(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	data := DepositData{
		PublicKey:             "validator-1",
		WithdrawalCredentials: "withdrawal-1",
		Amount:                32000000000,
		Signature:             "sig",
	}
	_, _, err := state.ProcessDeposit(blockTestDeposit(0, data, "0xabc123"))
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	state.SlashValidator(0)

	block := BeaconBlock{
		Slot: 65,
		Body: BeaconBlockBody{
			Eth1Data: Eth1Data{
				DepositRoot:  state.DepositRoot,
				DepositCount: state.Eth1DepositIndex,
				BlockHash:    "0xeth1block5",
			},
			Slashings: []SlashingOperation{
				{
					ValidatorIndex: 0,
					Reason:         "duplicate-slash",
				},
			},
		},
	}

	if err := state.ProcessBlock(block, 32); err == nil {
		t.Fatalf("expected duplicate slashing to fail")
	}
}
