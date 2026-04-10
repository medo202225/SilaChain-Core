package staking

import "silachain/pkg/types"

type Reward struct {
	Validator   types.Address `json:"validator"`
	BlockHeight uint64        `json:"block_height"`
	Amount      uint64        `json:"amount"`
	Reason      string        `json:"reason"`
}
