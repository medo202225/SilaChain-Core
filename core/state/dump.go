package state

type DumpAccount struct {
	Address  string
	Balance  uint64
	Nonce    uint64
	CodeHash string
	Storage  map[string]string
	Dirty    bool
	Deleted  bool
	Suicide  bool
	Touched  bool
}

type Dump struct {
	Accounts map[string]DumpAccount
}

func (s *StateDB) RawDump() Dump {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := Dump{
		Accounts: make(map[string]DumpAccount, len(s.stateObjects)),
	}
	for address, obj := range s.stateObjects {
		if obj == nil {
			continue
		}
		storage := make(map[string]string, len(obj.originStorage))
		for key, value := range obj.originStorage {
			storage[key] = value
		}
		out.Accounts[address] = DumpAccount{
			Address:  obj.address,
			Balance:  obj.data.Balance,
			Nonce:    obj.data.Nonce,
			CodeHash: obj.codeHash,
			Storage:  storage,
			Dirty:    obj.dirty,
			Deleted:  obj.deleted,
			Suicide:  obj.suicide,
			Touched:  obj.touched,
		}
	}
	return out
}
