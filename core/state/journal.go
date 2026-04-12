package state

type journalEntry interface {
	revert(*StateDB)
}

type balanceChange struct {
	account string
	prev    uint64
}

func (ch balanceChange) revert(s *StateDB) {
	if s == nil {
		return
	}
	obj := s.getStateObject(ch.account)
	if obj == nil {
		return
	}
	obj.data.Balance = ch.prev
	obj.dirty = true
	s.markStateObjectDirty(ch.account)
}

type nonceChange struct {
	account string
	prev    uint64
}

func (ch nonceChange) revert(s *StateDB) {
	if s == nil {
		return
	}
	obj := s.getStateObject(ch.account)
	if obj == nil {
		return
	}
	obj.data.Nonce = ch.prev
	obj.dirty = true
	s.markStateObjectDirty(ch.account)
}

type storageChange struct {
	account string
	key     string
	prev    string
	hadPrev bool
}

func (ch storageChange) revert(s *StateDB) {
	if s == nil {
		return
	}
	obj := s.getStateObject(ch.account)
	if obj == nil {
		return
	}
	if !ch.hadPrev {
		delete(obj.dirtyStorage, ch.key)
		delete(obj.pendingStorage, ch.key)
		if obj.originStorage != nil {
			delete(obj.originStorage, ch.key)
		}
		obj.dirty = true
		s.markStateObjectDirty(ch.account)
		return
	}
	obj.dirtyStorage[ch.key] = ch.prev
	obj.pendingStorage[ch.key] = ch.prev
	if obj.originStorage == nil {
		obj.originStorage = make(map[string]string)
	}
	obj.originStorage[ch.key] = ch.prev
	obj.dirty = true
	s.markStateObjectDirty(ch.account)
}

type createObjectChange struct {
	account string
}

func (ch createObjectChange) revert(s *StateDB) {
	if s == nil {
		return
	}
	delete(s.stateObjects, ch.account)
	delete(s.stateObjectsDirty, ch.account)
}

type refundChange struct {
	prev uint64
}

func (ch refundChange) revert(s *StateDB) {
	if s == nil {
		return
	}
	s.refund = ch.prev
}

type addLogChange struct {
	prevLen int
}

func (ch addLogChange) revert(s *StateDB) {
	if s == nil {
		return
	}
	if ch.prevLen < 0 {
		ch.prevLen = 0
	}
	if ch.prevLen > len(s.logs) {
		ch.prevLen = len(s.logs)
	}
	s.logs = s.logs[:ch.prevLen]
}

type accessListAddAddressChange struct {
	address string
}

func (ch accessListAddAddressChange) revert(s *StateDB) {
	if s == nil || s.accessList == nil {
		return
	}
	s.accessList.DeleteAddress(ch.address)
}

type accessListAddSlotChange struct {
	address string
	slot    string
}

func (ch accessListAddSlotChange) revert(s *StateDB) {
	if s == nil || s.accessList == nil {
		return
	}
	s.accessList.DeleteSlot(ch.address, ch.slot)
}

type transientStorageChange struct {
	address string
	key     string
	prev    string
	hadPrev bool
}

func (ch transientStorageChange) revert(s *StateDB) {
	if s == nil || s.transientStorage == nil {
		return
	}
	if !ch.hadPrev {
		s.transientStorage.Delete(ch.address, ch.key)
		return
	}
	s.transientStorage.Set(ch.address, ch.key, ch.prev)
}

type touchChange struct {
	account string
	prev    bool
}

func (ch touchChange) revert(s *StateDB) {
	if s == nil {
		return
	}
	obj := s.getStateObject(ch.account)
	if obj == nil {
		return
	}
	obj.touched = ch.prev
}

type codeChange struct {
	account  string
	prevCode []byte
	prevHash string
}

func (ch codeChange) revert(s *StateDB) {
	if s == nil {
		return
	}
	obj := s.getStateObject(ch.account)
	if obj == nil {
		return
	}
	obj.code = cloneBytes(ch.prevCode)
	obj.codeHash = ch.prevHash
	obj.dirty = true
	s.markStateObjectDirty(ch.account)
}

type suicideChange struct {
	account string
	prev    bool
}

func (ch suicideChange) revert(s *StateDB) {
	if s == nil {
		return
	}
	obj := s.getStateObject(ch.account)
	if obj == nil {
		return
	}
	obj.suicide = ch.prev
	obj.dirty = true
	s.markStateObjectDirty(ch.account)
}

type deleteChange struct {
	account string
	prev    bool
}

func (ch deleteChange) revert(s *StateDB) {
	if s == nil {
		return
	}
	obj := s.getStateObject(ch.account)
	if obj == nil {
		return
	}
	obj.deleted = ch.prev
	obj.dirty = true
	s.markStateObjectDirty(ch.account)
}

type journal struct {
	entries []journalEntry
}

func newJournal() *journal {
	return &journal{
		entries: make([]journalEntry, 0),
	}
}

func (j *journal) append(entry journalEntry) {
	if j == nil {
		return
	}
	j.entries = append(j.entries, entry)
}

func (j *journal) length() int {
	if j == nil {
		return 0
	}
	return len(j.entries)
}

func (j *journal) revert(s *StateDB, snapshot int) {
	if j == nil {
		return
	}
	if snapshot < 0 {
		snapshot = 0
	}
	if snapshot > len(j.entries) {
		snapshot = len(j.entries)
	}
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		j.entries[i].revert(s)
	}
	j.entries = j.entries[:snapshot]
}
