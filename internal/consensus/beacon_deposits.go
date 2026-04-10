package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

func (s *BeaconStateV1) GetValidatorIndexByPubKey(pubKey string) (int, bool) {
	if s == nil {
		return -1, false
	}

	for i, v := range s.Validators {
		if v.PublicKey == pubKey {
			return i, true
		}
	}

	return -1, false
}

func hashDepositData(data DepositData) string {
	payload := fmt.Sprintf(
		"%s|%s|%d|%s",
		data.PublicKey,
		data.WithdrawalCredentials,
		data.Amount,
		data.Signature,
	)
	sum := sha256.Sum256([]byte(payload))
	return "0x" + hex.EncodeToString(sum[:])
}

func hashPair(left string, right string) string {
	payload := strings.TrimSpace(left) + "|" + strings.TrimSpace(right)
	sum := sha256.Sum256([]byte(payload))
	return "0x" + hex.EncodeToString(sum[:])
}

func computeDepositRoot(dep Deposit) (string, error) {
	if len(dep.Proof) == 0 {
		return "", fmt.Errorf("deposit proof is required")
	}
	branch := dep.Proof[0]
	if len(branch) == 0 {
		return "", fmt.Errorf("deposit proof first branch is empty")
	}

	node := hashDepositData(dep.Data)
	index := dep.Index

	for _, sibling := range branch {
		sibling = strings.TrimSpace(sibling)
		if sibling == "" {
			return "", fmt.Errorf("deposit proof branch contains empty sibling")
		}

		if index%2 == 0 {
			node = hashPair(node, sibling)
		} else {
			node = hashPair(sibling, node)
		}
		index = index / 2
	}

	return node, nil
}

func validateDepositProof(dep Deposit) error {
	if dep.Data.PublicKey == "" {
		return fmt.Errorf("deposit public key is required")
	}
	if dep.Data.Amount == 0 {
		return fmt.Errorf("deposit amount must be > 0")
	}
	if strings.TrimSpace(dep.Root) == "" {
		return fmt.Errorf("deposit root is required")
	}

	computedRoot, err := computeDepositRoot(dep)
	if err != nil {
		return err
	}
	if computedRoot != strings.TrimSpace(dep.Root) {
		return fmt.Errorf("invalid deposit proof root")
	}

	return nil
}

func (s *BeaconStateV1) refreshValidatorBalance(index int) {
	if s == nil {
		return
	}
	if index < 0 || index >= len(s.Validators) || index >= len(s.Balances) {
		return
	}

	s.Validators[index].EffectiveBalance = normalizeEffectiveBalance(s.Balances[index])
}

func (s *BeaconStateV1) updateActivationEligibility(index int) {
	if s == nil {
		return
	}
	if index < 0 || index >= len(s.Validators) {
		return
	}

	v := &s.Validators[index]
	if v.ActivationEligibilityEpoch != 0 {
		return
	}
	if v.EffectiveBalance < maxEffectiveBalance {
		return
	}

	v.ActivationEligibilityEpoch = s.Epoch
}

func (s *BeaconStateV1) AddValidatorFromDeposit(data DepositData) (int, error) {
	if s == nil {
		return -1, fmt.Errorf("nil beacon state")
	}
	if data.PublicKey == "" {
		return -1, fmt.Errorf("deposit public key is required")
	}

	validator := ValidatorRecord{
		PublicKey:                  data.PublicKey,
		WithdrawalCredentials:      data.WithdrawalCredentials,
		EffectiveBalance:           normalizeEffectiveBalance(data.Amount),
		Slashed:                    false,
		ActivationEligibilityEpoch: 0,
		ActivationEpoch:            farFutureEpoch,
		ExitEpoch:                  farFutureEpoch,
		WithdrawableEpoch:          farFutureEpoch,
	}

	s.Validators = append(s.Validators, validator)
	s.Balances = append(s.Balances, data.Amount)

	index := len(s.Validators) - 1
	s.refreshValidatorBalance(index)
	s.updateActivationEligibility(index)

	return index, nil
}

func (s *BeaconStateV1) ApplyDeposit(data DepositData) (int, bool, error) {
	if s == nil {
		return -1, false, fmt.Errorf("nil beacon state")
	}
	if data.PublicKey == "" {
		return -1, false, fmt.Errorf("deposit public key is required")
	}
	if data.Amount == 0 {
		return -1, false, fmt.Errorf("deposit amount must be > 0")
	}

	index, exists := s.GetValidatorIndexByPubKey(data.PublicKey)
	if !exists {
		index, err := s.AddValidatorFromDeposit(data)
		if err != nil {
			return -1, false, err
		}
		return index, true, nil
	}

	s.Balances[index] += data.Amount
	if s.Validators[index].WithdrawalCredentials == "" && data.WithdrawalCredentials != "" {
		s.Validators[index].WithdrawalCredentials = data.WithdrawalCredentials
	}

	s.refreshValidatorBalance(index)
	s.updateActivationEligibility(index)

	return index, false, nil
}

func (s *BeaconStateV1) ProcessDeposit(dep Deposit) (int, bool, error) {
	if s == nil {
		return -1, false, fmt.Errorf("nil beacon state")
	}
	if dep.Index != s.Eth1DepositIndex {
		return -1, false, fmt.Errorf("unexpected deposit index %d, expected %d", dep.Index, s.Eth1DepositIndex)
	}
	if err := validateDepositProof(dep); err != nil {
		return -1, false, err
	}

	index, created, err := s.ApplyDeposit(dep.Data)
	if err != nil {
		return -1, false, err
	}

	s.DepositRoot = dep.Root
	s.Eth1DepositIndex++
	return index, created, nil
}
