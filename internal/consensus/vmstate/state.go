package vmstate

import "errors"

var (
	ErrNilState      = errors.New("vmstate: nil state")
	ErrEmptyAddress  = errors.New("vmstate: empty address")
	ErrEmptyStoreKey = errors.New("vmstate: empty storage key")
)

type Account struct {
	Address string            `json:"address"`
	Nonce   uint64            `json:"nonce"`
	Balance uint64            `json:"balance"`
	Code    []byte            `json:"code"`
	Storage map[string]string `json:"storage"`
}

func (a Account) HasCode() bool {
	return len(a.Code) > 0
}

func (a Account) StorageSlots() int {
	return len(a.Storage)
}

type State struct {
	accounts map[string]*Account
}

func New() *State {
	return &State{
		accounts: make(map[string]*Account),
	}
}

func (s *State) EnsureAccount(address string) (*Account, error) {
	if s == nil {
		return nil, ErrNilState
	}
	if address == "" {
		return nil, ErrEmptyAddress
	}

	acct, ok := s.accounts[address]
	if ok {
		return acct, nil
	}

	acct = &Account{
		Address: address,
		Nonce:   0,
		Balance: 0,
		Code:    nil,
		Storage: make(map[string]string),
	}
	s.accounts[address] = acct
	return acct, nil
}

func (s *State) GetAccount(address string) (Account, bool) {
	if s == nil || address == "" {
		return Account{}, false
	}
	acct, ok := s.accounts[address]
	if !ok {
		return Account{}, false
	}

	out := Account{
		Address: acct.Address,
		Nonce:   acct.Nonce,
		Balance: acct.Balance,
	}
	if len(acct.Code) > 0 {
		out.Code = append([]byte(nil), acct.Code...)
	}
	out.Storage = make(map[string]string, len(acct.Storage))
	for k, v := range acct.Storage {
		out.Storage[k] = v
	}
	return out, true
}

func (s *State) SetNonce(address string, nonce uint64) error {
	acct, err := s.EnsureAccount(address)
	if err != nil {
		return err
	}
	acct.Nonce = nonce
	return nil
}

func (s *State) AddBalance(address string, amount uint64) error {
	acct, err := s.EnsureAccount(address)
	if err != nil {
		return err
	}
	acct.Balance += amount
	return nil
}

func (s *State) SetBalance(address string, amount uint64) error {
	acct, err := s.EnsureAccount(address)
	if err != nil {
		return err
	}
	acct.Balance = amount
	return nil
}

func (s *State) SetCode(address string, code []byte) error {
	acct, err := s.EnsureAccount(address)
	if err != nil {
		return err
	}
	acct.Code = append([]byte(nil), code...)
	return nil
}

func (s *State) GetCode(address string) ([]byte, bool) {
	acct, ok := s.GetAccount(address)
	if !ok {
		return nil, false
	}
	return append([]byte(nil), acct.Code...), true
}

func (s *State) SetStorage(address, key, value string) error {
	if key == "" {
		return ErrEmptyStoreKey
	}
	acct, err := s.EnsureAccount(address)
	if err != nil {
		return err
	}
	acct.Storage[key] = value
	return nil
}

func (s *State) GetStorage(address, key string) (string, bool) {
	if s == nil || address == "" || key == "" {
		return "", false
	}
	acct, ok := s.accounts[address]
	if !ok {
		return "", false
	}
	val, ok := acct.Storage[key]
	return val, ok
}

func (s *State) ClearStorage(address string) error {
	acct, err := s.EnsureAccount(address)
	if err != nil {
		return err
	}
	acct.Storage = make(map[string]string)
	return nil
}
