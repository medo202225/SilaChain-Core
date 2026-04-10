package staking

import "silachain/pkg/types"

type Slash struct {
	Validator types.Address `json:"validator"`
	Amount    uint64        `json:"amount"`
	Reason    string        `json:"reason"`
}
