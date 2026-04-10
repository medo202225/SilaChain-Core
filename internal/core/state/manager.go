package state

import (
	"silachain/internal/accounts"
	"silachain/pkg/types"
)

type Manager struct {
	accounts *accounts.Manager
}

func NewManager(accountManager *accounts.Manager) *Manager {
	return &Manager{
		accounts: accountManager,
	}
}

func (m *Manager) Accounts() *accounts.Manager {
	return m.accounts
}

func (m *Manager) GetAccount(address types.Address) (*accounts.Account, error) {
	acc, ok := m.accounts.Get(address)
	if !ok {
		return nil, accounts.ErrAccountNotFound
	}
	return acc, nil
}

func (m *Manager) GetBalance(address types.Address) (types.Amount, error) {
	acc, err := m.GetAccount(address)
	if err != nil {
		return 0, err
	}
	return acc.Balance, nil
}
