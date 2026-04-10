package state

import "silachain/pkg/types"

type Checkpoint struct {
	Height    types.Height `json:"height"`
	StateRoot types.Hash   `json:"state_root"`
}
