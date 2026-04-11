package state

import "sync"

type StateDB struct {
	mu       sync.RWMutex
	accounts map[string]*Account
}

func NewStateDB() *StateDB {
	return &StateDB{
		accounts: make(map[string]*Account),
	}
}

func (s *StateDB) EnsureAccount(address string) *Account {
	s.mu.Lock()
	defer s.mu.Unlock()

	acc, ok := s.accounts[address]
	if ok {
		return acc
	}

	acc = &Account{
		Address: address,
		Balance: 0,
		Nonce:   0,
	}
	s.accounts[address] = acc
	return acc
}

func (s *StateDB) SetBalance(address string, balance uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	acc, ok := s.accounts[address]
	if !ok {
		acc = &Account{
			Address: address,
			Balance: 0,
			Nonce:   0,
		}
		s.accounts[address] = acc
	}
	acc.Balance = balance
}

func (s *StateDB) GetBalance(address string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acc, ok := s.accounts[address]
	if !ok {
		return 0
	}
	return acc.Balance
}

func (s *StateDB) GetNonce(address string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acc, ok := s.accounts[address]
	if !ok {
		return 0
	}
	return acc.Nonce
}

func (s *StateDB) SetNonce(address string, nonce uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	acc, ok := s.accounts[address]
	if !ok {
		acc = &Account{
			Address: address,
			Balance: 0,
			Nonce:   0,
		}
		s.accounts[address] = acc
	}
	acc.Nonce = nonce
}

func (s *StateDB) AccountNonce(address string) uint64 {
	return s.GetNonce(address)
}

func (s *StateDB) SnapshotAccounts() map[string]Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]Account, len(s.accounts))
	for address, account := range s.accounts {
		if account == nil {
			continue
		}
		out[address] = *account
	}
	return out
}
