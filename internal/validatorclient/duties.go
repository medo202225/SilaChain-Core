package validatorclient

import "context"

type ProposalDuty struct {
	PublicKey   string `json:"public_key"`
	Slot        uint64 `json:"slot"`
	SigningRoot []byte `json:"signing_root"`
}

type AttestationDuty struct {
	PublicKey   string `json:"public_key"`
	SourceEpoch uint64 `json:"source_epoch"`
	TargetEpoch uint64 `json:"target_epoch"`
	SigningRoot []byte `json:"signing_root"`
}

type DutySigner interface {
	SignProposal(ctx context.Context, duty ProposalDuty) (*SignatureResult, error)
	SignAttestation(ctx context.Context, duty AttestationDuty) (*SignatureResult, error)
}
