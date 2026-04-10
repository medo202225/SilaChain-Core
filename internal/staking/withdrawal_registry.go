package staking

import "silachain/pkg/types"

type WithdrawalRegistry struct {
	items []Withdrawal
}

func NewWithdrawalRegistry() *WithdrawalRegistry {
	return &WithdrawalRegistry{
		items: []Withdrawal{},
	}
}

func (r *WithdrawalRegistry) Add(item Withdrawal) {
	if r == nil {
		return
	}
	r.items = append(r.items, item)
}

func (r *WithdrawalRegistry) Load(items []Withdrawal) {
	if r == nil {
		return
	}
	r.items = make([]Withdrawal, len(items))
	copy(r.items, items)
}

func (r *WithdrawalRegistry) All() []Withdrawal {
	if r == nil {
		return nil
	}
	out := make([]Withdrawal, len(r.items))
	copy(out, r.items)
	return out
}

func (r *WithdrawalRegistry) TotalForAddress(address types.Address) uint64 {
	if r == nil {
		return 0
	}
	var total uint64
	for _, w := range r.items {
		if w.Address == address {
			total += w.Amount
		}
	}
	return total
}
