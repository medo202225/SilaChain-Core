package staking

import "silachain/pkg/types"

type JailRegistry struct {
	items map[string]Jail
}

func NewJailRegistry() *JailRegistry {
	return &JailRegistry{
		items: map[string]Jail{},
	}
}

func (r *JailRegistry) Set(validator types.Address, jailed bool, reason string) {
	if r == nil {
		return
	}
	r.items[string(validator)] = Jail{
		Validator: validator,
		Jailed:    jailed,
		Reason:    reason,
	}
}

func (r *JailRegistry) IsJailed(validator types.Address) bool {
	if r == nil {
		return false
	}
	item, ok := r.items[string(validator)]
	if !ok {
		return false
	}
	return item.Jailed
}

func (r *JailRegistry) Load(items []Jail) {
	if r == nil {
		return
	}
	r.items = map[string]Jail{}
	for _, item := range items {
		r.items[string(item.Validator)] = item
	}
}

func (r *JailRegistry) All() []Jail {
	if r == nil {
		return nil
	}
	out := make([]Jail, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out
}
