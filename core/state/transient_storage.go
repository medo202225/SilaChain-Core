package state

type TransientStorage struct {
	data map[string]map[string]string
}

func NewTransientStorage() *TransientStorage {
	return &TransientStorage{
		data: make(map[string]map[string]string),
	}
}

func (ts *TransientStorage) Set(address, key, value string) {
	if ts == nil || address == "" || key == "" {
		return
	}
	if ts.data[address] == nil {
		ts.data[address] = make(map[string]string)
	}
	ts.data[address][key] = value
}

func (ts *TransientStorage) Get(address, key string) (string, bool) {
	if ts == nil {
		return "", false
	}
	account := ts.data[address]
	if account == nil {
		return "", false
	}
	v, ok := account[key]
	return v, ok
}

func (ts *TransientStorage) Delete(address, key string) {
	if ts == nil {
		return
	}
	account := ts.data[address]
	if account == nil {
		return
	}
	delete(account, key)
	if len(account) == 0 {
		delete(ts.data, address)
	}
}

func (ts *TransientStorage) Reset() {
	if ts == nil {
		return
	}
	ts.data = make(map[string]map[string]string)
}

func (ts *TransientStorage) Copy() *TransientStorage {
	if ts == nil {
		return NewTransientStorage()
	}
	out := NewTransientStorage()
	for address, account := range ts.data {
		dst := make(map[string]string, len(account))
		for key, value := range account {
			dst[key] = value
		}
		out.data[address] = dst
	}
	return out
}
