package execution

import (
	"fmt"

	coretypes "silachain/internal/core/types"
)

// CANONICAL OWNERSHIP: legacy proposal executor shim only.
// Canonical proposal execution lives in the consensus/execution pipeline.

type ProposalExecutor struct{}

func NewProposalExecutor(_ any) *ProposalExecutor {
	return &ProposalExecutor{}
}

func (e *ProposalExecutor) ExecuteProposal(_ any) (*coretypes.Block, error) {
	_ = e
	return nil, fmt.Errorf("legacy proposal executor is disabled; use the consensus/execution pipeline")
}
