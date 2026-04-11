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

func (s *StateDB) ensureAccount(address string) *Account {
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

	acc := s.ensureAccount(address)
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

func (s *StateDB) AccountNonce(address string) uint64 {
	return s.GetNonce(address)
}
