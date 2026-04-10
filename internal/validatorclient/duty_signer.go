package validatorclient

import (
	"context"
	"fmt"

	"silachain/internal/validatorclient/slashing"
)

type ProtectedDutySigner struct {
	loaded *LoadedVotingKeystore
	store  slashing.Store
}

func NewProtectedDutySigner(loaded *LoadedVotingKeystore, store slashing.Store) (*ProtectedDutySigner, error) {
	if loaded == nil {
		return nil, fmt.Errorf("loaded voting keystore is nil")
	}
	if store == nil {
		return nil, fmt.Errorf("slashing store is nil")
	}

	return &ProtectedDutySigner{
		loaded: loaded,
		store:  store,
	}, nil
}

func (s *ProtectedDutySigner) SignProposal(ctx context.Context, duty ProposalDuty) (*SignatureResult, error) {
	if s == nil {
		return nil, fmt.Errorf("protected duty signer is nil")
	}
	return ProtectedSignBlock(ctx, s.loaded, s.store, duty.Slot, duty.SigningRoot)
}

func (s *ProtectedDutySigner) SignAttestation(ctx context.Context, duty AttestationDuty) (*SignatureResult, error) {
	if s == nil {
		return nil, fmt.Errorf("protected duty signer is nil")
	}
	return ProtectedSignAttestation(ctx, s.loaded, s.store, duty.SourceEpoch, duty.TargetEpoch, duty.SigningRoot)
}
