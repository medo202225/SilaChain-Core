package staking

import "silachain/pkg/types"

type Entry struct {
	Validator types.Address `json:"validator"`
	Stake     uint64        `json:"stake"`
}

type Registry struct {
	entries map[string]Entry
}

func NewRegistry() *Registry {
	return &Registry{
		entries: map[string]Entry{},
	}
}

func (r *Registry) Set(validator types.Address, stake uint64) {
	if r == nil {
		return
	}
	r.entries[string(validator)] = Entry{
		Validator: validator,
		Stake:     stake,
	}
}

func (r *Registry) Get(validator types.Address) (Entry, bool) {
	if r == nil {
		return Entry{}, false
	}
	e, ok := r.entries[string(validator)]
	return e, ok
}

func (r *Registry) Load(entries []Entry) {
	if r == nil {
		return
	}
	r.entries = map[string]Entry{}
	for _, e := range entries {
		r.entries[string(e.Validator)] = e
	}
}

func (r *Registry) All() []Entry {
	if r == nil {
		return nil
	}
	out := make([]Entry, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e)
	}
	return out
}
