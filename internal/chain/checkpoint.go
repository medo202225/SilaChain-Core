package chain

import "silachain/pkg/types"

type Checkpoint struct {
	Height types.Height `json:"height"`
	Hash   types.Hash   `json:"hash"`
}
