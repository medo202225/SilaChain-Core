package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

const (
	activationChurnLimit    uint64 = 1
	minWithdrawabilityDelay uint64 = 2
)

func hashBeaconRoot(label string, v uint64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	sum := sha256.Sum256(append([]byte(label+":"), buf...))
	return "0x" + hex.EncodeToString(sum[:])
}

func (s *BeaconStateV1) AdvanceSlot(slot uint64, slotsPerEpoch uint64) {
	if s == nil {
		return
	}
	if slotsPerEpoch == 0 {
		slotsPerEpoch = 32
	}

	previousEpoch := s.Epoch

	s.Slot = slot
	s.Epoch = slot / slotsPerEpoch
	s.HeadBlockRoot = hashBeaconRoot("head", slot)

	if slot == 0 {
		s.SafeBlockRoot = s.HeadBlockRoot
	} else {
		s.SafeBlockRoot = hashBeaconRoot("safe", slot-1)
	}

	if s.Epoch != previousEpoch {
		s.ProcessEpochTransition()
	}

	s.updateCheckpoints(previousEpoch)
}

func (s *BeaconStateV1) ProcessEpochTransition() {
	if s == nil {
		return
	}

	s.ProcessRegistryUpdates()
	s.ProcessSlashingWithdrawals()
}

func (s *BeaconStateV1) ProcessRegistryUpdates() {
	if s == nil {
		return
	}

	pending := s.PendingActivationIndices()
	if len(pending) == 0 {
		return
	}

	var activated uint64
	for _, index := range pending {
		if activated >= activationChurnLimit {
			break
		}

		v := &s.Validators[index]
		if v.ActivationEligibilityEpoch > s.Epoch {
			continue
		}
		if v.ActivationEpoch != farFutureEpoch {
			continue
		}

		v.ActivationEpoch = s.Epoch + 1
		activated++
	}
}

func (s *BeaconStateV1) ProcessSlashingWithdrawals() {
	if s == nil {
		return
	}

	for i := range s.Validators {
		v := &s.Validators[i]
		if !v.Slashed {
			continue
		}
		if v.WithdrawableEpoch == farFutureEpoch {
			continue
		}
		if s.Epoch < v.WithdrawableEpoch {
			continue
		}
		if i >= len(s.Balances) {
			continue
		}

		s.Balances[i] = 0
		s.refreshValidatorBalance(i)
	}
}

func (s *BeaconStateV1) InitiateValidatorExit(index int) {
	if s == nil {
		return
	}
	if index < 0 || index >= len(s.Validators) {
		return
	}

	v := &s.Validators[index]
	if v.ExitEpoch != farFutureEpoch {
		return
	}

	exitEpoch := s.Epoch + 1
	v.ExitEpoch = exitEpoch

	withdrawableEpoch := exitEpoch + minWithdrawabilityDelay
	if v.WithdrawableEpoch == farFutureEpoch || v.WithdrawableEpoch < withdrawableEpoch {
		v.WithdrawableEpoch = withdrawableEpoch
	}
}

func (s *BeaconStateV1) SlashValidator(index int) {
	if s == nil {
		return
	}
	if index < 0 || index >= len(s.Validators) {
		return
	}

	v := &s.Validators[index]
	if v.Slashed {
		return
	}

	s.InitiateValidatorExit(index)
	v.Slashed = true

	slashedBalance := s.Balances[index] / 32
	if slashedBalance == 0 && s.Balances[index] > 0 {
		slashedBalance = 1
	}
	if slashedBalance > s.Balances[index] {
		slashedBalance = s.Balances[index]
	}

	s.Balances[index] -= slashedBalance
	s.refreshValidatorBalance(index)

	minSlashWithdrawableEpoch := s.Epoch + minWithdrawabilityDelay
	if v.WithdrawableEpoch < minSlashWithdrawableEpoch {
		v.WithdrawableEpoch = minSlashWithdrawableEpoch
	}
}

func (s *BeaconStateV1) updateCheckpoints(previousEpoch uint64) {
	if s == nil {
		return
	}

	if s.Epoch == 0 {
		s.CurrentJustifiedCheckpoint = Checkpoint{
			Epoch: 0,
			Root:  s.HeadBlockRoot,
		}
		s.FinalizedCheckpoint = Checkpoint{
			Epoch: 0,
			Root:  s.HeadBlockRoot,
		}
		return
	}

	if s.Epoch != previousEpoch {
		justifiedEpoch := s.Epoch - 1
		s.CurrentJustifiedCheckpoint = Checkpoint{
			Epoch: justifiedEpoch,
			Root:  hashBeaconRoot("justified", justifiedEpoch),
		}

		finalizedEpoch := uint64(0)
		if s.Epoch > 1 {
			finalizedEpoch = s.Epoch - 2
		}
		s.FinalizedCheckpoint = Checkpoint{
			Epoch: finalizedEpoch,
			Root:  hashBeaconRoot("finalized", finalizedEpoch),
		}
	}
}
