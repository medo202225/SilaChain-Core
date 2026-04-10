package consensus

import "context"

type ForkchoiceNotifier interface {
	ForkchoiceUpdatedV1(headBlockHash string, safeBlockHash string, finalizedBlockHash string) (map[string]any, error)
}

type SlotAwareForkchoiceNotifier interface {
	NotifyForkchoice(ctx context.Context, state *BeaconStateV1) error
}
