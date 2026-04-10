package staking

import "silachain/pkg/types"

type UnbondClaim struct {
	Address types.Address `json:"address"`
	Amount  uint64        `json:"amount"`
	Reason  string        `json:"reason"`
}
