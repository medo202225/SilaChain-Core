package validator

// CANONICAL OWNERSHIP: validator domain package for member/set/registry and legacy-loaded signing key contracts used by chain, storage, and legacy consensus paths.

import "silachain/pkg/types"

type Registry struct {
	members []Member
}

func NewRegistry() *Registry {
	return &Registry{
		members: []Member{},
	}
}

func (r *Registry) LoadFromSet(set *Set) {
	if r == nil || set == nil {
		return
	}
	r.members = make([]Member, len(set.Members))
	copy(r.members, set.Members)
}

func (r *Registry) Members() []Member {
	if r == nil {
		return nil
	}
	out := make([]Member, len(r.members))
	copy(out, r.members)
	return out
}

func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.members)
}

func (r *Registry) ProposerAt(index int) (types.Address, bool) {
	if r == nil || len(r.members) == 0 {
		return "", false
	}
	if index < 0 {
		return "", false
	}
	return r.members[index%len(r.members)].Address, true
}
