package consensus

import (
	"context"
	"fmt"
	"log"
	"strconv"
)

type EngineForkchoiceNotifier struct {
	client *EngineClient
	store  *ForkchoiceStore
}

func NewEngineForkchoiceNotifier(client *EngineClient) (*EngineForkchoiceNotifier, error) {
	if client == nil {
		return nil, fmt.Errorf("engine client is nil")
	}
	return &EngineForkchoiceNotifier{
		client: client,
		store:  NewForkchoiceStore(),
	}, nil
}

func (n *EngineForkchoiceNotifier) NotifyForkchoice(ctx context.Context, state *BeaconStateV1) error {
	_ = ctx
	if n == nil || n.client == nil {
		return fmt.Errorf("engine forkchoice notifier is nil")
	}
	if state == nil {
		return fmt.Errorf("beacon state is nil")
	}
	if n.store == nil {
		return fmt.Errorf("forkchoice store is nil")
	}

	if err := n.store.UpdateFromBeaconState(state); err != nil {
		return err
	}

	fc := n.store.State()

	payloadAttrs := map[string]any{
		"timestamp":             strconv.FormatUint(state.Slot, 10),
		"prevRandao":            fc.HeadRoot,
		"suggestedFeeRecipient": "0x0000000000000000000000000000000000000000",
	}

	resp, err := n.client.ForkchoiceUpdatedV1(
		fc.HeadRoot,
		fc.SafeRoot,
		fc.FinalizedRoot,
		payloadAttrs,
	)
	if err != nil {
		return err
	}

	parsed, err := parseForkchoiceUpdatedResult(resp)
	if err != nil {
		return err
	}

	switch parsed.PayloadStatus.Status {
	case "VALID", "ACCEPTED", "SYNCING":
	default:
		return fmt.Errorf(
			"forkchoiceUpdated returned non-accepted status=%s latest_valid_hash=%s validation_error=%s",
			parsed.PayloadStatus.Status,
			parsed.PayloadStatus.LatestValidHash,
			parsed.PayloadStatus.ValidationError,
		)
	}

	if parsed.PayloadID != "" {
		state.LatestPayloadID = parsed.PayloadID
	}

	log.Printf(
		"consensus engine forkchoiceUpdatedV1 ok: slot=%d epoch=%d head=%s safe=%s finalized=%s payload_status=%s latest_valid_hash=%s payload_id=%s response=%v",
		state.Slot,
		state.Epoch,
		fc.HeadRoot,
		fc.SafeRoot,
		fc.FinalizedRoot,
		parsed.PayloadStatus.Status,
		parsed.PayloadStatus.LatestValidHash,
		state.LatestPayloadID,
		resp,
	)

	return nil
}
