package staking

import "silachain/pkg/types"

type Withdrawal struct {
	Address types.Address `json:"address"`
	Amount  uint64        `json:"amount"`
	Reason  string        `json:"reason"`
}
