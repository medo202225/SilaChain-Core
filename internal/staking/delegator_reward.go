package staking

import "silachain/pkg/types"

type DelegatorReward struct {
	Delegator   types.Address `json:"delegator"`
	Validator   types.Address `json:"validator"`
	BlockHeight uint64        `json:"block_height"`
	Amount      uint64        `json:"amount"`
	Reason      string        `json:"reason"`
}
