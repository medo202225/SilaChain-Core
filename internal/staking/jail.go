package staking

import "silachain/pkg/types"

type Jail struct {
	Validator types.Address `json:"validator"`
	Jailed    bool          `json:"jailed"`
	Reason    string        `json:"reason"`
}
