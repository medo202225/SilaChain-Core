package staking

type UndelegationRegistry struct {
	items []Undelegation
}

func NewUndelegationRegistry() *UndelegationRegistry {
	return &UndelegationRegistry{
		items: []Undelegation{},
	}
}

func (r *UndelegationRegistry) Add(item Undelegation) {
	if r == nil {
		return
	}
	r.items = append(r.items, item)
}

func (r *UndelegationRegistry) Load(items []Undelegation) {
	if r == nil {
		return
	}
	r.items = make([]Undelegation, len(items))
	copy(r.items, items)
}

func (r *UndelegationRegistry) All() []Undelegation {
	if r == nil {
		return nil
	}
	out := make([]Undelegation, len(r.items))
	copy(out, r.items)
	return out
}
