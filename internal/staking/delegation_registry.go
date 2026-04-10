package staking

import "silachain/pkg/types"

type DelegationRegistry struct {
	items map[string]Delegation
}

func NewDelegationRegistry() *DelegationRegistry {
	return &DelegationRegistry{
		items: map[string]Delegation{},
	}
}

func delegationKey(delegator types.Address, validator types.Address) string {
	return string(delegator) + "->" + string(validator)
}

func (r *DelegationRegistry) Set(delegator types.Address, validator types.Address, amount uint64) {
	if r == nil {
		return
	}
	key := delegationKey(delegator, validator)
	r.items[key] = Delegation{
		Delegator: delegator,
		Validator: validator,
		Amount:    amount,
	}
}

func (r *DelegationRegistry) Get(delegator types.Address, validator types.Address) (Delegation, bool) {
	if r == nil {
		return Delegation{}, false
	}
	d, ok := r.items[delegationKey(delegator, validator)]
	return d, ok
}

func (r *DelegationRegistry) Delete(delegator types.Address, validator types.Address) {
	if r == nil {
		return
	}
	delete(r.items, delegationKey(delegator, validator))
}

func (r *DelegationRegistry) Load(items []Delegation) {
	if r == nil {
		return
	}
	r.items = map[string]Delegation{}
	for _, d := range items {
		r.items[delegationKey(d.Delegator, d.Validator)] = d
	}
}

func (r *DelegationRegistry) All() []Delegation {
	if r == nil {
		return nil
	}
	out := make([]Delegation, 0, len(r.items))
	for _, d := range r.items {
		out = append(out, d)
	}
	return out
}

func (r *DelegationRegistry) TotalForValidator(validator types.Address) uint64 {
	if r == nil {
		return 0
	}

	var total uint64
	for _, d := range r.items {
		if d.Validator == validator {
			total += d.Amount
		}
	}
	return total
}

func (r *DelegationRegistry) ForValidator(validator types.Address) []Delegation {
	if r == nil {
		return nil
	}

	out := make([]Delegation, 0)
	for _, d := range r.items {
		if d.Validator == validator {
			out = append(out, d)
		}
	}
	return out
}
