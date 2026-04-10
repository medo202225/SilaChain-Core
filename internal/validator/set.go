package validator

// CANONICAL OWNERSHIP: validator domain package for member/set/registry and legacy-loaded signing key contracts used by chain, storage, and legacy consensus paths.

import "silachain/pkg/types"

type Member struct {
	Address   types.Address `json:"address"`
	PublicKey string        `json:"public_key"`
	Power     uint64        `json:"power"`
	Stake     uint64        `json:"stake"`
}

type Set struct {
	Members []Member
}

func NewSet(members []Member) *Set {
	return &Set{
		Members: members,
	}
}

func (s *Set) ProposerAt(index int) (types.Address, bool) {
	if s == nil || len(s.Members) == 0 {
		return "", false
	}
	if index < 0 {
		return "", false
	}
	return s.Members[index%len(s.Members)].Address, true
}

func (s *Set) Len() int {
	if s == nil {
		return 0
	}
	return len(s.Members)
}

func (s *Set) All() []Member {
	if s == nil {
		return nil
	}
	out := make([]Member, len(s.Members))
	copy(out, s.Members)
	return out
}
