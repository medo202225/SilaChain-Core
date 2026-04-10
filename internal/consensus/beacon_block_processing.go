package consensus

import "fmt"

type VoluntaryExit struct {
	ValidatorIndex uint64 `json:"validator_index"`
	Epoch          uint64 `json:"epoch"`
	Signature      string `json:"signature"`
}

type SlashingOperation struct {
	ValidatorIndex uint64 `json:"validator_index"`
	Reason         string `json:"reason"`
}

type BeaconBlockBody struct {
	Eth1Data       Eth1Data            `json:"eth1_data"`
	Deposits       []Deposit           `json:"deposits"`
	VoluntaryExits []VoluntaryExit     `json:"voluntary_exits"`
	Slashings      []SlashingOperation `json:"slashings"`
}

type BeaconBlock struct {
	Slot uint64          `json:"slot"`
	Body BeaconBlockBody `json:"body"`
}

func (s *BeaconStateV1) ProcessDeposits(deposits []Deposit) error {
	if s == nil {
		return fmt.Errorf("nil beacon state")
	}

	for _, dep := range deposits {
		if _, _, err := s.ProcessDeposit(dep); err != nil {
			return err
		}
	}

	return nil
}

func (s *BeaconStateV1) ProcessEth1Data(eth1 Eth1Data, deposits []Deposit) error {
	if s == nil {
		return fmt.Errorf("nil beacon state")
	}

	expectedCount := s.Eth1DepositIndex + uint64(len(deposits))
	if eth1.DepositCount != expectedCount {
		return fmt.Errorf("invalid eth1 deposit count %d, expected %d", eth1.DepositCount, expectedCount)
	}

	if len(deposits) == 0 {
		if eth1.DepositRoot != s.DepositRoot {
			return fmt.Errorf("eth1 deposit root %s does not match current state deposit root %s", eth1.DepositRoot, s.DepositRoot)
		}
		s.Eth1Data = eth1
		return nil
	}

	lastDepositRoot := deposits[len(deposits)-1].Root
	if eth1.DepositRoot != lastDepositRoot {
		return fmt.Errorf("eth1 deposit root %s does not match last deposit root %s", eth1.DepositRoot, lastDepositRoot)
	}

	s.Eth1Data = eth1
	return nil
}

func (s *BeaconStateV1) ProcessVoluntaryExits(exits []VoluntaryExit) error {
	if s == nil {
		return fmt.Errorf("nil beacon state")
	}

	for _, exit := range exits {
		index := int(exit.ValidatorIndex)
		if !s.CanValidatorVoluntarilyExit(index, exit.Epoch) {
			return fmt.Errorf("validator %d cannot voluntarily exit at epoch %d", exit.ValidatorIndex, exit.Epoch)
		}
		s.InitiateValidatorExit(index)
	}

	return nil
}

func (s *BeaconStateV1) ProcessSlashings(slashings []SlashingOperation) error {
	if s == nil {
		return fmt.Errorf("nil beacon state")
	}

	for _, slashing := range slashings {
		index := int(slashing.ValidatorIndex)
		if !s.CanSlashValidator(index) {
			return fmt.Errorf("validator %d cannot be slashed", slashing.ValidatorIndex)
		}
		s.SlashValidator(index)
	}

	return nil
}

func (s *BeaconStateV1) ProcessBlock(block BeaconBlock, slotsPerEpoch uint64) error {
	if s == nil {
		return fmt.Errorf("nil beacon state")
	}

	if block.Slot < s.Slot {
		return fmt.Errorf("block slot %d is behind current slot %d", block.Slot, s.Slot)
	}

	s.AdvanceSlot(block.Slot, slotsPerEpoch)

	if err := s.ProcessEth1Data(block.Body.Eth1Data, block.Body.Deposits); err != nil {
		return err
	}
	if err := s.ProcessDeposits(block.Body.Deposits); err != nil {
		return err
	}
	if err := s.ProcessVoluntaryExits(block.Body.VoluntaryExits); err != nil {
		return err
	}
	if err := s.ProcessSlashings(block.Body.Slashings); err != nil {
		return err
	}

	return nil
}
