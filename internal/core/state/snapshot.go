package state

import "silachain/pkg/types"

type Snapshot struct {
	Height    types.Height `json:"height"`
	StateRoot types.Hash   `json:"state_root"`
}
