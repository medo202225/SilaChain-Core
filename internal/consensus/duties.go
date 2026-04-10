package consensus

import (
	"context"

	validatorclient "silachain/internal/validatorclient"
)

type DutyProvider interface {
	NextProposalDuty(ctx context.Context) (*validatorclient.ProposalDuty, bool, error)
	NextAttestationDuty(ctx context.Context) (*validatorclient.AttestationDuty, bool, error)
}

type SlotAwareDutyProvider interface {
	DutyProvider
	AdvanceToSlot(slot uint64)
}

type NoopDutyProvider struct{}

func (n *NoopDutyProvider) NextProposalDuty(ctx context.Context) (*validatorclient.ProposalDuty, bool, error) {
	_ = ctx
	return nil, false, nil
}

func (n *NoopDutyProvider) NextAttestationDuty(ctx context.Context) (*validatorclient.AttestationDuty, bool, error) {
	_ = ctx
	return nil, false, nil
}

func (n *NoopDutyProvider) AdvanceToSlot(slot uint64) {
	_ = slot
}
