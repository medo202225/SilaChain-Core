package state

import "silachain/pkg/types"

type JournalSnapshot struct {
	ID            int
	ContractCode  map[types.Address]string
	ContractStore map[types.Address]map[string]string
}

type Journal struct {
	nextID    int
	snapshots []JournalSnapshot
}

func NewJournal() *Journal {
	return &Journal{
		nextID:    1,
		snapshots: make([]JournalSnapshot, 0),
	}
}

func (j *Journal) CreateSnapshot(
	codeRegistry *ContractCodeRegistry,
	storage *ContractStorage,
) int {
	if j == nil {
		return -1
	}

	id := j.nextID
	j.nextID++

	var codeCopy map[types.Address]string
	if codeRegistry != nil {
		codeCopy = codeRegistry.All()
	} else {
		codeCopy = make(map[types.Address]string)
	}

	var storageCopy map[types.Address]map[string]string
	if storage != nil {
		storageCopy = storage.All()
	} else {
		storageCopy = make(map[types.Address]map[string]string)
	}

	j.snapshots = append(j.snapshots, JournalSnapshot{
		ID:            id,
		ContractCode:  codeCopy,
		ContractStore: storageCopy,
	})

	return id
}

func (j *Journal) Commit(id int) {
	if j == nil {
		return
	}

	index := j.indexOf(id)
	if index < 0 {
		return
	}

	j.snapshots = j.snapshots[:index]
}

func (j *Journal) Revert(
	id int,
	codeRegistry *ContractCodeRegistry,
	storage *ContractStorage,
) bool {
	if j == nil {
		return false
	}

	index := j.indexOf(id)
	if index < 0 {
		return false
	}

	snapshot := j.snapshots[index]

	if codeRegistry != nil {
		codeRegistry.Load(snapshot.ContractCode)
	}

	if storage != nil {
		storage.Load(snapshot.ContractStore)
	}

	j.snapshots = j.snapshots[:index]
	return true
}

func (j *Journal) Len() int {
	if j == nil {
		return 0
	}
	return len(j.snapshots)
}

func (j *Journal) indexOf(id int) int {
	for i := len(j.snapshots) - 1; i >= 0; i-- {
		if j.snapshots[i].ID == id {
			return i
		}
	}
	return -1
}
