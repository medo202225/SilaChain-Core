package consensus

import (
	"context"
	"testing"
)

type testForkchoiceNotifier struct {
	called    bool
	payloadID string
}

func (n *testForkchoiceNotifier) NotifyForkchoice(ctx context.Context, state *BeaconStateV1) error {
	_ = ctx
	n.called = true
	if state != nil && n.payloadID != "" {
		state.LatestPayloadID = n.payloadID
	}
	return nil
}

type testProposalExecutor struct {
	called bool
}

func (e *testProposalExecutor) ProposeBlock(ctx context.Context) error {
	_ = ctx
	e.called = true
	return nil
}

func TestTransitionCoordinatorRunForkchoiceAndMaybeProposeFalse(t *testing.T) {
	notifier := &testForkchoiceNotifier{payloadID: "payload-123"}
	executor := &testProposalExecutor{}
	coordinator := NewTransitionCoordinator(notifier, executor)

	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	if err := coordinator.RunForkchoiceAndMaybePropose(context.Background(), state, false); err != nil {
		t.Fatalf("transition coordinator failed: %v", err)
	}

	if !notifier.called {
		t.Fatalf("expected forkchoice notifier to be called")
	}
	if executor.called {
		t.Fatalf("did not expect proposal executor to be called")
	}
}

func TestTransitionCoordinatorRunForkchoiceAndMaybeProposeTrue(t *testing.T) {
	notifier := &testForkchoiceNotifier{payloadID: "payload-123"}
	executor := &testProposalExecutor{}
	coordinator := NewTransitionCoordinator(notifier, executor)

	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	if err := coordinator.RunForkchoiceAndMaybePropose(context.Background(), state, true); err != nil {
		t.Fatalf("transition coordinator failed: %v", err)
	}

	if !notifier.called {
		t.Fatalf("expected forkchoice notifier to be called")
	}
	if !executor.called {
		t.Fatalf("expected proposal executor to be called")
	}
}
