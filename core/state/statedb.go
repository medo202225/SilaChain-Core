package state

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type revision struct {
	id           int
	journalIndex int
}

type StateDB struct {
	mu                sync.RWMutex
	db                Database
	journal           *journal
	stateObjects      map[string]*stateObject
	stateObjectsDirty map[string]struct{}
	revisions         []revision
	nextRevisionID    int

	refund           uint64
	logs             []StateLog
	accessList       *AccessList
	transientStorage *TransientStorage
	accessEvents     *AccessEvents
}

func NewStateDB() *StateDB {
	return NewStateDBWithDatabase(NewMemoryDatabase())
}

func NewStateDBWithDatabase(db Database) *StateDB {
	if db == nil {
		db = NewMemoryDatabase()
	}
	return &StateDB{
		db:                db,
		journal:           newJournal(),
		stateObjects:      make(map[string]*stateObject),
		stateObjectsDirty: make(map[string]struct{}),
		revisions:         make([]revision, 0),
		logs:              make([]StateLog, 0),
		accessList:        NewAccessList(),
		transientStorage:  NewTransientStorage(),
		accessEvents:      NewAccessEvents(),
	}
}

func (s *StateDB) getStateObject(address string) *stateObject {
	if s == nil {
		return nil
	}
	return s.stateObjects[address]
}

func (s *StateDB) markStateObjectDirty(address string) {
	if s == nil {
		return
	}
	s.stateObjectsDirty[address] = struct{}{}
}

func (s *StateDB) createObject(address string) *stateObject {
	if s == nil {
		return nil
	}
	obj := newStateObject(s, address, Account{Address: address})
	if tr, err := s.db.OpenStorageTrie(address); err == nil {
		obj.trie = tr
	}
	s.stateObjects[address] = obj
	s.markStateObjectDirty(address)
	if s.journal != nil {
		s.journal.append(createObjectChange{account: address})
	}
	return obj
}

func (s *StateDB) getOrNewStateObject(address string) *stateObject {
	if s == nil {
		return nil
	}
	if obj := s.getStateObject(address); obj != nil {
		return obj
	}
	return s.createObject(address)
}

func (s *StateDB) EnsureAccount(address string) *Account {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj := s.getOrNewStateObject(address)
	if obj == nil {
		return nil
	}
	return &obj.data
}

func (s *StateDB) SetBalance(address string, balance uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj := s.getOrNewStateObject(address)
	if obj == nil {
		return
	}
	obj.SetBalance(balance)
}

func (s *StateDB) AddBalance(address string, amount uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj := s.getOrNewStateObject(address)
	if obj == nil {
		return
	}
	obj.AddBalance(amount)
}

func (s *StateDB) SubBalance(address string, amount uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj := s.getOrNewStateObject(address)
	if obj == nil {
		return
	}
	obj.SubBalance(amount)
}

func (s *StateDB) GetBalance(address string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj := s.getStateObject(address)
	if obj == nil {
		return 0
	}
	return obj.Balance()
}

func (s *StateDB) GetNonce(address string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj := s.getStateObject(address)
	if obj == nil {
		return 0
	}
	return obj.Nonce()
}

func (s *StateDB) SetNonce(address string, nonce uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj := s.getOrNewStateObject(address)
	if obj == nil {
		return
	}
	obj.SetNonce(nonce)
}

func (s *StateDB) GetState(address, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj := s.getStateObject(address)
	if obj == nil {
		return "", false
	}
	return obj.GetState(key)
}

func (s *StateDB) SetState(address, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj := s.getOrNewStateObject(address)
	if obj == nil {
		return
	}
	obj.SetState(key, value)
}

func (s *StateDB) GetCode(address string) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj := s.getStateObject(address)
	if obj == nil {
		return nil
	}
	return obj.Code()
}

func (s *StateDB) GetCodeHash(address string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj := s.getStateObject(address)
	if obj == nil {
		return ""
	}
	return obj.CodeHash()
}

func (s *StateDB) SetCode(address string, code []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj := s.getOrNewStateObject(address)
	if obj == nil {
		return
	}
	obj.SetCode(code)
}

func (s *StateDB) AddRefund(amount uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.journal != nil {
		s.journal.append(refundChange{prev: s.refund})
	}
	s.refund += amount
}

func (s *StateDB) SubRefund(amount uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.journal != nil {
		s.journal.append(refundChange{prev: s.refund})
	}
	if amount >= s.refund {
		s.refund = 0
		return
	}
	s.refund -= amount
}

func (s *StateDB) GetRefund() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.refund
}

func (s *StateDB) AddLog(log StateLog) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.journal != nil {
		s.journal.append(addLogChange{prevLen: len(s.logs)})
	}
	s.logs = append(s.logs, log)
}

func (s *StateDB) Logs() []StateLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]StateLog, len(s.logs))
	copy(out, s.logs)
	return out
}

func (s *StateDB) AddAddressToAccessList(address string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.accessList == nil {
		s.accessList = NewAccessList()
	}
	if !s.accessList.ContainsAddress(address) && s.journal != nil {
		s.journal.append(accessListAddAddressChange{address: address})
	}
	s.accessList.AddAddress(address)
}

func (s *StateDB) AddSlotToAccessList(address, slot string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.accessList == nil {
		s.accessList = NewAccessList()
	}
	_, slotPresent := s.accessList.Contains(address, slot)
	if !slotPresent && s.journal != nil {
		s.journal.append(accessListAddSlotChange{
			address: address,
			slot:    slot,
		})
	}
	s.accessList.AddSlot(address, slot)
}

func (s *StateDB) AddressInAccessList(address string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.accessList == nil {
		return false
	}
	return s.accessList.ContainsAddress(address)
}

func (s *StateDB) SlotInAccessList(address, slot string) (bool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.accessList == nil {
		return false, false
	}
	return s.accessList.Contains(address, slot)
}

func (s *StateDB) SetTransientState(address, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.transientStorage == nil {
		s.transientStorage = NewTransientStorage()
	}
	prev, hadPrev := s.transientStorage.Get(address, key)
	if s.journal != nil {
		s.journal.append(transientStorageChange{
			address: address,
			key:     key,
			prev:    prev,
			hadPrev: hadPrev,
		})
	}
	s.transientStorage.Set(address, key, value)
}

func (s *StateDB) GetTransientState(address, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.transientStorage == nil {
		return "", false
	}
	return s.transientStorage.Get(address, key)
}

func (s *StateDB) Prepare(rules any, sender, coinbase string, to *string, precompiles []string, accessListEntries []AccessTuple) {
	s.AddAddressToAccessList(sender)
	if coinbase != "" {
		s.AddAddressToAccessList(coinbase)
	}
	if to != nil {
		s.AddAddressToAccessList(*to)
	}
	for _, precompile := range precompiles {
		s.AddAddressToAccessList(precompile)
	}
	for _, entry := range accessListEntries {
		s.AddAddressToAccessList(entry.Address)
		for _, slot := range entry.StorageKeys {
			s.AddSlotToAccessList(entry.Address, slot)
		}
	}
	if s.transientStorage != nil {
		s.transientStorage.Reset()
	}
	if s.accessEvents != nil {
		s.accessEvents.Reset()
	}
	_ = rules
}

func (s *StateDB) Exist(address string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.stateObjects[address]
	return ok
}

func (s *StateDB) Empty(address string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj := s.getStateObject(address)
	if obj == nil {
		return true
	}
	return obj.Empty()
}

func (s *StateDB) Touch(address string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj := s.getOrNewStateObject(address)
	if obj == nil {
		return
	}
	obj.Touch()
}

func (s *StateDB) AccountNonce(address string) uint64 {
	return s.GetNonce(address)
}

func (s *StateDB) Snapshot() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextRevisionID
	s.nextRevisionID++

	journalIndex := 0
	if s.journal != nil {
		journalIndex = s.journal.length()
	}
	s.revisions = append(s.revisions, revision{
		id:           id,
		journalIndex: journalIndex,
	})
	return id
}

func (s *StateDB) RevertToSnapshot(snapshot int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	revIndex := -1
	journalIndex := 0
	for i := len(s.revisions) - 1; i >= 0; i-- {
		if s.revisions[i].id == snapshot {
			revIndex = i
			journalIndex = s.revisions[i].journalIndex
			break
		}
	}
	if revIndex < 0 {
		return
	}

	if s.journal != nil {
		s.journal.revert(s, journalIndex)
	}
	s.revisions = s.revisions[:revIndex]
	s.rebuildDirtySet()
}

func (s *StateDB) rebuildDirtySet() {
	s.stateObjectsDirty = make(map[string]struct{})
	for address, obj := range s.stateObjects {
		if obj == nil {
			continue
		}
		if obj.IsDirty() || obj.HasSuicided() || obj.IsDeleted() {
			s.stateObjectsDirty[address] = struct{}{}
		}
	}
}

func (s *StateDB) Finalise(deleteEmptyObjects bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for address := range s.stateObjectsDirty {
		obj := s.stateObjects[address]
		if obj == nil {
			continue
		}

		if deleteEmptyObjects && obj.Empty() {
			obj.MarkDeleted()
		}

		if obj.IsDeleted() {
			delete(s.stateObjects, address)
			continue
		}

		if obj.trie == nil {
			if tr, err := s.db.OpenStorageTrie(address); err == nil {
				obj.trie = tr
			}
		}
		if obj.trie != nil {
			for key, value := range obj.pendingStorage {
				if value == "" {
					obj.trie.DeleteStorage(key)
				} else {
					obj.trie.UpdateStorage(key, value)
				}
			}
		}

		obj.originStorage = cloneStorageMap(obj.pendingStorage)
		obj.pendingStorage = make(map[string]string)
		obj.dirtyStorage = make(map[string]string)
		obj.dirty = false
	}

	s.stateObjectsDirty = make(map[string]struct{})
}

func (s *StateDB) IntermediateRoot(deleteEmptyObjects bool) string {
	s.Finalise(deleteEmptyObjects)

	s.mu.RLock()
	defer s.mu.RUnlock()

	addresses := make([]string, 0, len(s.stateObjects))
	for address := range s.stateObjects {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)

	parts := make([]string, 0, len(addresses))
	for _, address := range addresses {
		obj := s.stateObjects[address]
		if obj == nil || obj.IsDeleted() {
			continue
		}
		parts = append(parts, encodeStateObjectForRoot(obj))
	}
	return fmt.Sprintf("state-root-%s", strings.Join(parts, "|"))
}

func (s *StateDB) Commit(deleteEmptyObjects bool) (string, error) {
	root := s.IntermediateRoot(deleteEmptyObjects)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.revisions = nil
	if s.journal != nil {
		s.journal.entries = nil
	}
	return root, nil
}

func (s *StateDB) SnapshotAccounts() map[string]Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]Account, len(s.stateObjects))
	for address, obj := range s.stateObjects {
		if obj == nil {
			continue
		}
		out[address] = obj.Account()
	}
	return out
}

func cloneStorageMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func encodeStateObjectForRoot(obj *stateObject) string {
	if obj == nil {
		return ""
	}

	keys := make([]string, 0, len(obj.originStorage))
	for key := range obj.originStorage {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	storageParts := make([]string, 0, len(keys))
	for _, key := range keys {
		storageParts = append(storageParts, key+"="+obj.originStorage[key])
	}

	return fmt.Sprintf(
		"%s:%d:%d:%s:%s",
		obj.Address(),
		obj.Balance(),
		obj.Nonce(),
		obj.CodeHash(),
		strings.Join(storageParts, ","),
	)
}
