package accounts

import "silachain/pkg/types"

func NewRegistry() *Manager {
	return NewManager()
}

func MustGetAccount(m *Manager, address types.Address) *Account {
	acc, ok := m.Get(address)
	if !ok {
		panic(ErrAccountNotFound)
	}
	return acc
}
