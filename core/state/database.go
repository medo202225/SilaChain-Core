package state

type Database interface {
	OpenStorageTrie(address string) (StorageTrie, error)
	CopyTrie(StorageTrie) StorageTrie
}

type StorageTrie interface {
	GetStorage(key string) (string, bool)
	UpdateStorage(key, value string)
	DeleteStorage(key string)
	Copy() StorageTrie
}

type MemoryStorageTrie struct {
	data map[string]string
}

func NewMemoryStorageTrie() *MemoryStorageTrie {
	return &MemoryStorageTrie{
		data: make(map[string]string),
	}
}

func (t *MemoryStorageTrie) GetStorage(key string) (string, bool) {
	if t == nil {
		return "", false
	}
	v, ok := t.data[key]
	return v, ok
}

func (t *MemoryStorageTrie) UpdateStorage(key, value string) {
	if t == nil {
		return
	}
	t.data[key] = value
}

func (t *MemoryStorageTrie) DeleteStorage(key string) {
	if t == nil {
		return
	}
	delete(t.data, key)
}

func (t *MemoryStorageTrie) Copy() StorageTrie {
	if t == nil {
		return NewMemoryStorageTrie()
	}
	out := NewMemoryStorageTrie()
	for k, v := range t.data {
		out.data[k] = v
	}
	return out
}

type MemoryDatabase struct {
	storage map[string]*MemoryStorageTrie
}

func NewMemoryDatabase() *MemoryDatabase {
	return &MemoryDatabase{
		storage: make(map[string]*MemoryStorageTrie),
	}
}

func (db *MemoryDatabase) OpenStorageTrie(address string) (StorageTrie, error) {
	if db == nil {
		return NewMemoryStorageTrie(), nil
	}
	tr, ok := db.storage[address]
	if !ok {
		tr = NewMemoryStorageTrie()
		db.storage[address] = tr
	}
	return tr, nil
}

func (db *MemoryDatabase) CopyTrie(tr StorageTrie) StorageTrie {
	if tr == nil {
		return NewMemoryStorageTrie()
	}
	return tr.Copy()
}
