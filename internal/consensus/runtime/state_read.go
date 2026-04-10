package runtime

import "errors"

var (
	ErrNilRuntimeStateRead = errors.New("runtime: nil runtime state read")
	ErrNilStateRead        = errors.New("runtime: nil state read")
)

type AccountResult struct {
	Address      string `json:"address"`
	Nonce        uint64 `json:"nonce"`
	Balance      uint64 `json:"balance"`
	HasCode      bool   `json:"hasCode"`
	StorageSlots int    `json:"storageSlots"`
	Found        bool   `json:"found"`
}

type StateCodeResult struct {
	Address string `json:"address"`
	Code    []byte `json:"code"`
	Found   bool   `json:"found"`
}

type StateStorageResult struct {
	Address string `json:"address"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Found   bool   `json:"found"`
}

func (r *Runtime) StateAccount(address string) (AccountResult, error) {
	if r == nil {
		return AccountResult{}, ErrNilRuntimeStateRead
	}
	if r.state == nil {
		return AccountResult{}, ErrNilStateRead
	}
	if address == "" {
		return AccountResult{
			Address: "",
			Found:   false,
		}, nil
	}

	acct, ok := r.state.Account(address)
	if !ok {
		return AccountResult{
			Address: address,
			Found:   false,
		}, nil
	}

	return AccountResult{
		Address:      acct.Address,
		Nonce:        acct.Nonce,
		Balance:      acct.Balance,
		HasCode:      acct.HasCode(),
		StorageSlots: acct.StorageSlots(),
		Found:        true,
	}, nil
}

func (r *Runtime) StateCode(address string) (StateCodeResult, error) {
	if r == nil {
		return StateCodeResult{}, ErrNilRuntimeStateRead
	}
	if r.state == nil {
		return StateCodeResult{}, ErrNilStateRead
	}

	code, ok := r.state.Code(address)
	if !ok {
		return StateCodeResult{
			Address: address,
			Found:   false,
		}, nil
	}

	return StateCodeResult{
		Address: address,
		Code:    code,
		Found:   true,
	}, nil
}

func (r *Runtime) StateStorage(address, key string) (StateStorageResult, error) {
	if r == nil {
		return StateStorageResult{}, ErrNilRuntimeStateRead
	}
	if r.state == nil {
		return StateStorageResult{}, ErrNilStateRead
	}

	value, ok := r.state.Storage(address, key)
	if !ok {
		return StateStorageResult{
			Address: address,
			Key:     key,
			Found:   false,
		}, nil
	}

	return StateStorageResult{
		Address: address,
		Key:     key,
		Value:   value,
		Found:   true,
	}, nil
}
