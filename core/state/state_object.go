package state

import "fmt"

type Account struct {
	Address string
	Balance uint64
	Nonce   uint64
}

type stateObject struct {
	address string
	data    Account
	db      *StateDB

	trie StorageTrie

	originStorage  map[string]string
	pendingStorage map[string]string
	dirtyStorage   map[string]string

	code     []byte
	codeHash string

	dirty   bool
	deleted bool
	suicide bool
	touched bool
}

func newStateObject(db *StateDB, address string, data Account) *stateObject {
	if data.Address == "" {
		data.Address = address
	}
	return &stateObject{
		address:        address,
		data:           data,
		db:             db,
		originStorage:  make(map[string]string),
		pendingStorage: make(map[string]string),
		dirtyStorage:   make(map[string]string),
		code:           nil,
		codeHash:       "",
	}
}

func (obj *stateObject) Address() string {
	if obj == nil {
		return ""
	}
	return obj.address
}

func (obj *stateObject) Balance() uint64 {
	if obj == nil {
		return 0
	}
	return obj.data.Balance
}

func (obj *stateObject) Nonce() uint64 {
	if obj == nil {
		return 0
	}
	return obj.data.Nonce
}

func (obj *stateObject) SetBalance(amount uint64) {
	if obj == nil {
		return
	}
	if obj.db != nil && obj.db.journal != nil {
		obj.db.journal.append(balanceChange{
			account: obj.address,
			prev:    obj.data.Balance,
		})
	}
	obj.data.Balance = amount
	obj.markDirty()
}

func (obj *stateObject) AddBalance(amount uint64) {
	if obj == nil || amount == 0 {
		return
	}
	obj.SetBalance(obj.data.Balance + amount)
}

func (obj *stateObject) SubBalance(amount uint64) {
	if obj == nil || amount == 0 {
		return
	}
	if amount >= obj.data.Balance {
		obj.SetBalance(0)
		return
	}
	obj.SetBalance(obj.data.Balance - amount)
}

func (obj *stateObject) SetNonce(nonce uint64) {
	if obj == nil {
		return
	}
	if obj.db != nil && obj.db.journal != nil {
		obj.db.journal.append(nonceChange{
			account: obj.address,
			prev:    obj.data.Nonce,
		})
	}
	obj.data.Nonce = nonce
	obj.markDirty()
}

func (obj *stateObject) GetState(key string) (string, bool) {
	if obj == nil {
		return "", false
	}
	if v, ok := obj.dirtyStorage[key]; ok {
		return v, true
	}
	if v, ok := obj.pendingStorage[key]; ok {
		return v, true
	}
	if v, ok := obj.originStorage[key]; ok {
		return v, true
	}
	if obj.trie != nil {
		return obj.trie.GetStorage(key)
	}
	return "", false
}

func (obj *stateObject) SetState(key, value string) {
	if obj == nil {
		return
	}
	prev, hadPrev := obj.GetState(key)
	if obj.db != nil && obj.db.journal != nil {
		obj.db.journal.append(storageChange{
			account: obj.address,
			key:     key,
			prev:    prev,
			hadPrev: hadPrev,
		})
	}

	obj.dirtyStorage[key] = value
	obj.pendingStorage[key] = value
	if obj.originStorage == nil {
		obj.originStorage = make(map[string]string)
	}
	obj.originStorage[key] = value
	obj.markDirty()
}

func (obj *stateObject) SetCode(code []byte) {
	if obj == nil {
		return
	}
	if obj.db != nil && obj.db.journal != nil {
		obj.db.journal.append(codeChange{
			account:  obj.address,
			prevCode: cloneBytes(obj.code),
			prevHash: obj.codeHash,
		})
	}
	obj.code = cloneBytes(code)
	obj.codeHash = simpleCodeHash(code)
	obj.markDirty()
}

func (obj *stateObject) Code() []byte {
	if obj == nil {
		return nil
	}
	return cloneBytes(obj.code)
}

func (obj *stateObject) CodeHash() string {
	if obj == nil {
		return ""
	}
	return obj.codeHash
}

func (obj *stateObject) Touch() {
	if obj == nil {
		return
	}
	prev := obj.touched
	if obj.db != nil && obj.db.journal != nil {
		obj.db.journal.append(touchChange{
			account: obj.address,
			prev:    prev,
		})
	}
	obj.touched = true
	obj.markDirty()
}

func (obj *stateObject) markDirty() {
	if obj == nil {
		return
	}
	obj.dirty = true
	if obj.db != nil {
		obj.db.markStateObjectDirty(obj.address)
	}
}

func (obj *stateObject) IsDirty() bool {
	if obj == nil {
		return false
	}
	return obj.dirty
}

func (obj *stateObject) Suicide() {
	if obj == nil {
		return
	}
	prev := obj.suicide
	if obj.db != nil && obj.db.journal != nil {
		obj.db.journal.append(suicideChange{
			account: obj.address,
			prev:    prev,
		})
	}
	obj.suicide = true
	obj.markDirty()
}

func (obj *stateObject) HasSuicided() bool {
	if obj == nil {
		return false
	}
	return obj.suicide
}

func (obj *stateObject) MarkDeleted() {
	if obj == nil {
		return
	}
	prev := obj.deleted
	if obj.db != nil && obj.db.journal != nil {
		obj.db.journal.append(deleteChange{
			account: obj.address,
			prev:    prev,
		})
	}
	obj.deleted = true
	obj.markDirty()
}

func (obj *stateObject) IsDeleted() bool {
	if obj == nil {
		return false
	}
	return obj.deleted
}

func (obj *stateObject) IsTouched() bool {
	if obj == nil {
		return false
	}
	return obj.touched
}

func (obj *stateObject) Empty() bool {
	if obj == nil {
		return true
	}
	return obj.data.Balance == 0 && obj.data.Nonce == 0 && len(obj.code) == 0
}

func (obj *stateObject) Account() Account {
	if obj == nil {
		return Account{}
	}
	return obj.data
}

func cloneBytes(in []byte) []byte {
	if in == nil {
		return nil
	}
	out := make([]byte, len(in))
	copy(out, in)
	return out
}

func simpleCodeHash(code []byte) string {
	if len(code) == 0 {
		return ""
	}
	return fmt.Sprintf("codehash-%d", len(code))
}
