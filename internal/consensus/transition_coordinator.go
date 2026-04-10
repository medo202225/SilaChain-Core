package consensus

// CANONICAL OWNERSHIP: root consensus package is limited to beacon state, scheduling, transition coordination, and validator coordination.
// Engine, engine API, forkchoice, runtime, txpool, executionstate, and p2p ownership live in their dedicated subpackages.

import (
	"context"
	"fmt"
)

type ProposalResultExecutor interface {
	ProposeBlockWithResult(ctx context.Context) (*ProposalTransitionResult, error)
}

type TransitionResult struct {
	ForkchoiceApplied bool                      `json:"forkchoice_applied"`
	ProposalApplied   bool                      `json:"proposal_applied"`
	PayloadID         string                    `json:"payload_id"`
	ProposalResult    *ProposalTransitionResult `json:"proposal_result"`
}

type TransitionCoordinator struct {
	forkchoiceNotifier SlotAwareForkchoiceNotifier
	proposalExecutor   ProposalExecutor
}

func NewTransitionCoordinator(
	forkchoiceNotifier SlotAwareForkchoiceNotifier,
	proposalExecutor ProposalExecutor,
) *TransitionCoordinator {
	return &TransitionCoordinator{
		forkchoiceNotifier: forkchoiceNotifier,
		proposalExecutor:   proposalExecutor,
	}
}

func (c *TransitionCoordinator) RunForkchoice(ctx context.Context, state *BeaconStateV1) error {
	if c == nil {
		return fmt.Errorf("transition coordinator is nil")
	}
	if c.forkchoiceNotifier == nil {
		return fmt.Errorf("forkchoice notifier is nil")
	}
	if state == nil {
		return fmt.Errorf("beacon state is nil")
	}

	return c.forkchoiceNotifier.NotifyForkchoice(ctx, state)
}

func (c *TransitionCoordinator) RunProposal(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("transition coordinator is nil")
	}
	if c.proposalExecutor == nil {
		return fmt.Errorf("proposal executor is nil")
	}

	return c.proposalExecutor.ProposeBlock(ctx)
}

func (c *TransitionCoordinator) RunForkchoiceAndMaybePropose(
	ctx context.Context,
	state *BeaconStateV1,
	shouldPropose bool,
) error {
	_, err := c.RunForkchoiceAndMaybeProposeWithResult(ctx, state, shouldPropose)
	return err
}

func (c *TransitionCoordinator) RunForkchoiceAndMaybeProposeWithResult(
	ctx context.Context,
	state *BeaconStateV1,
	shouldPropose bool,
) (*TransitionResult, error) {
	if c == nil {
		return nil, fmt.Errorf("transition coordinator is nil")
	}
	if state == nil {
		return nil, fmt.Errorf("beacon state is nil")
	}

	if err := c.RunForkchoice(ctx, state); err != nil {
		return nil, err
	}

	out := &TransitionResult{
		ForkchoiceApplied: true,
		PayloadID:         state.LatestPayloadID,
	}

	if !shouldPropose {
		return out, nil
	}

	if state.LatestPayloadID == "" {
		return nil, fmt.Errorf("missing payload id after forkchoice")
	}

	if execWithResult, ok := c.proposalExecutor.(ProposalResultExecutor); ok {
		proposalResult, err := execWithResult.ProposeBlockWithResult(ctx)
		if err != nil {
			return nil, err
		}
		if proposalResult == nil {
			return nil, fmt.Errorf("missing proposal result")
		}
		if proposalResult.PayloadID == "" {
			return nil, fmt.Errorf("missing proposal payload id")
		}
		if proposalResult.PayloadID != state.LatestPayloadID {
			return nil, fmt.Errorf("proposal payload id %s does not match forkchoice payload id %s", proposalResult.PayloadID, state.LatestPayloadID)
		}
		if !proposalResult.PayloadAccepted {
			return nil, fmt.Errorf("proposal payload was not accepted, status=%s", proposalResult.PayloadStatus)
		}

		out.ProposalApplied = true
		out.ProposalResult = proposalResult
		out.PayloadID = proposalResult.PayloadID
		return out, nil
	}

	if err := c.RunProposal(ctx); err != nil {
		return nil, err
	}
	out.ProposalApplied = true
	return out, nil
}
