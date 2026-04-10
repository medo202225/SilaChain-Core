package staking

import "silachain/pkg/types"

type Delegation struct {
	Delegator types.Address `json:"delegator"`
	Validator types.Address `json:"validator"`
	Amount    uint64        `json:"amount"`
}
