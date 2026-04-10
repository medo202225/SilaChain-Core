package consensus

import "fmt"

func validatorRecordToGenesisDeposit(index uint64, v ValidatorRecord) Deposit {
	data := DepositData{
		PublicKey:             v.PublicKey,
		WithdrawalCredentials: v.WithdrawalCredentials,
		Amount:                v.EffectiveBalance,
		Signature:             fmt.Sprintf("genesis-deposit-%d", index),
	}

	dep := Deposit{
		Index: index,
		Proof: [][]string{{fmt.Sprintf("0xgenesisproof%02d", index)}},
		Data:  data,
	}

	root, err := computeDepositRoot(dep)
	if err != nil {
		panic(err)
	}
	dep.Root = root

	return dep
}

func LoadBeaconStateFromDepositSource(path string, slotsPerEpoch uint64) (*BeaconStateV1, error) {
	validators, err := LoadBeaconValidatorsFromFile(path)
	if err != nil {
		return nil, err
	}

	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(0, slotsPerEpoch)

	for i, v := range validators {
		dep := validatorRecordToGenesisDeposit(uint64(i), v)
		if _, _, err := state.ProcessDeposit(dep); err != nil {
			return nil, fmt.Errorf("process genesis deposit %d failed: %w", i, err)
		}
	}

	state.Eth1Data = Eth1Data{
		DepositRoot:  state.DepositRoot,
		DepositCount: state.Eth1DepositIndex,
		BlockHash:    "0xgenesis",
	}

	return state, nil
}
