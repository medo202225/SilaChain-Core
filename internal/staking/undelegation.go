package staking

import "silachain/pkg/types"

type Undelegation struct {
	Delegator    types.Address `json:"delegator"`
	Validator    types.Address `json:"validator"`
	Amount       uint64        `json:"amount"`
	Reason       string        `json:"reason"`
	UnlockHeight uint64        `json:"unlock_height"`
}
