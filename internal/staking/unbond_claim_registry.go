package staking

import "silachain/pkg/types"

type UnbondClaimRegistry struct {
	items []UnbondClaim
}

func NewUnbondClaimRegistry() *UnbondClaimRegistry {
	return &UnbondClaimRegistry{
		items: []UnbondClaim{},
	}
}

func (r *UnbondClaimRegistry) Add(item UnbondClaim) {
	if r == nil {
		return
	}
	r.items = append(r.items, item)
}

func (r *UnbondClaimRegistry) Load(items []UnbondClaim) {
	if r == nil {
		return
	}
	r.items = make([]UnbondClaim, len(items))
	copy(r.items, items)
}

func (r *UnbondClaimRegistry) All() []UnbondClaim {
	if r == nil {
		return nil
	}
	out := make([]UnbondClaim, len(r.items))
	copy(out, r.items)
	return out
}

func (r *UnbondClaimRegistry) TotalForAddress(address types.Address) uint64 {
	if r == nil {
		return 0
	}
	var total uint64
	for _, c := range r.items {
		if c.Address == address {
			total += c.Amount
		}
	}
	return total
}
