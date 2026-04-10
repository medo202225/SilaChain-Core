package consensus

import (
	"context"
	"testing"
)

type proposalResultExecutor struct {
	called bool
	result *ProposalTransitionResult
}

func (e *proposalResultExecutor) ProposeBlock(ctx context.Context) error {
	_ = ctx
	e.called = true
	return nil
}

func (e *proposalResultExecutor) ProposeBlockWithResult(ctx context.Context) (*ProposalTransitionResult, error) {
	_ = ctx
	e.called = true
	if e.result != nil {
		return e.result, nil
	}
	return &ProposalTransitionResult{
		PayloadID: "payload-123",
		PayloadResponse: map[string]any{
			"result": "payload",
		},
		NewPayloadResp: map[string]any{
			"result": map[string]any{
				"status": "VALID",
			},
		},
		PayloadStatus:   "VALID",
		PayloadAccepted: true,
	}, nil
}

type payloadSettingForkchoiceNotifier struct {
	payloadID string
}

func (n *payloadSettingForkchoiceNotifier) NotifyForkchoice(ctx context.Context, state *BeaconStateV1) error {
	_ = ctx
	state.LatestPayloadID = n.payloadID
	return nil
}

func TestTransitionCoordinatorReturnsTransitionResultWithoutProposal(t *testing.T) {
	notifier := &payloadSettingForkchoiceNotifier{payloadID: "payload-from-forkchoice"}
	executor := &proposalResultExecutor{}
	coordinator := NewTransitionCoordinator(notifier, executor)

	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	result, err := coordinator.RunForkchoiceAndMaybeProposeWithResult(context.Background(), state, false)
	if err != nil {
		t.Fatalf("transition coordinator failed: %v", err)
	}

	if result == nil {
		t.Fatalf("expected transition result")
	}
	if !result.ForkchoiceApplied {
		t.Fatalf("expected forkchoice applied")
	}
	if result.ProposalApplied {
		t.Fatalf("did not expect proposal applied")
	}
	if result.PayloadID != "payload-from-forkchoice" {
		t.Fatalf("expected payload id payload-from-forkchoice, got %s", result.PayloadID)
	}
}

func TestTransitionCoordinatorReturnsTransitionResultWithProposal(t *testing.T) {
	notifier := &payloadSettingForkchoiceNotifier{payloadID: "payload-123"}
	executor := &proposalResultExecutor{}
	coordinator := NewTransitionCoordinator(notifier, executor)

	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	result, err := coordinator.RunForkchoiceAndMaybeProposeWithResult(context.Background(), state, true)
	if err != nil {
		t.Fatalf("transition coordinator failed: %v", err)
	}

	if result == nil {
		t.Fatalf("expected transition result")
	}
	if !result.ForkchoiceApplied {
		t.Fatalf("expected forkchoice applied")
	}
	if !result.ProposalApplied {
		t.Fatalf("expected proposal applied")
	}
	if result.ProposalResult == nil {
		t.Fatalf("expected proposal result")
	}
	if result.PayloadID != "payload-123" {
		t.Fatalf("expected payload id payload-123, got %s", result.PayloadID)
	}
	if result.ProposalResult.PayloadStatus != "VALID" {
		t.Fatalf("expected payload status VALID, got %s", result.ProposalResult.PayloadStatus)
	}
	if !result.ProposalResult.PayloadAccepted {
		t.Fatalf("expected payload to be accepted")
	}
}

func TestTransitionCoordinatorRejectsMissingPayloadIDAfterForkchoice(t *testing.T) {
	notifier := &payloadSettingForkchoiceNotifier{payloadID: ""}
	executor := &proposalResultExecutor{}
	coordinator := NewTransitionCoordinator(notifier, executor)

	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	_, err := coordinator.RunForkchoiceAndMaybeProposeWithResult(context.Background(), state, true)
	if err == nil {
		t.Fatalf("expected missing payload id after forkchoice to fail")
	}
}

func TestTransitionCoordinatorRejectsMismatchedPayloadID(t *testing.T) {
	notifier := &payloadSettingForkchoiceNotifier{payloadID: "payload-a"}
	executor := &proposalResultExecutor{
		result: &ProposalTransitionResult{
			PayloadID:       "payload-b",
			PayloadResponse: map[string]any{"result": "payload"},
			NewPayloadResp:  map[string]any{"result": map[string]any{"status": "VALID"}},
			PayloadStatus:   "VALID",
			PayloadAccepted: true,
		},
	}
	coordinator := NewTransitionCoordinator(notifier, executor)

	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	_, err := coordinator.RunForkchoiceAndMaybeProposeWithResult(context.Background(), state, true)
	if err == nil {
		t.Fatalf("expected mismatched payload id to fail")
	}
}

func TestTransitionCoordinatorRejectsUnacceptedPayload(t *testing.T) {
	notifier := &payloadSettingForkchoiceNotifier{payloadID: "payload-a"}
	executor := &proposalResultExecutor{
		result: &ProposalTransitionResult{
			PayloadID:       "payload-a",
			PayloadResponse: map[string]any{"result": "payload"},
			NewPayloadResp:  map[string]any{"result": map[string]any{"status": "INVALID"}},
			PayloadStatus:   "INVALID",
			PayloadAccepted: false,
		},
	}
	coordinator := NewTransitionCoordinator(notifier, executor)

	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	_, err := coordinator.RunForkchoiceAndMaybeProposeWithResult(context.Background(), state, true)
	if err == nil {
		t.Fatalf("expected unaccepted payload to fail")
	}
}
