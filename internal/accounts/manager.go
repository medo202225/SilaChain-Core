package accounts

import "silachain/pkg/types"

type Manager struct {
	accounts map[types.Address]*Account
}

func NewManager() *Manager {
	return &Manager{
		accounts: make(map[types.Address]*Account),
	}
}

func (m *Manager) Add(acc *Account) error {
	if acc == nil || acc.Address == "" {
		return ErrInvalidAccount
	}
	if _, exists := m.accounts[acc.Address]; exists {
		return ErrAccountAlreadyExists
	}
	m.accounts[acc.Address] = acc
	return nil
}

func (m *Manager) RegisterAccount(address types.Address, publicKey string) (*Account, error) {
	acc := NewAccount(address, publicKey)
	if err := m.Add(acc); err != nil {
		return nil, err
	}
	return acc, nil
}

func (m *Manager) Get(address types.Address) (*Account, bool) {
	acc, ok := m.accounts[address]
	return acc, ok
}

func (m *Manager) Exists(address types.Address) bool {
	_, ok := m.accounts[address]
	return ok
}

func (m *Manager) Set(acc *Account) error {
	if acc == nil || acc.Address == "" {
		return ErrInvalidAccount
	}
	m.accounts[acc.Address] = acc
	return nil
}

func (m *Manager) All() map[types.Address]*Account {
	out := make(map[types.Address]*Account, len(m.accounts))
	for k, v := range m.accounts {
		out[k] = v
	}
	return out
}
