// Copyright 2026 The SILA Authors
// This file is part of the sila-library.
//
// The sila-library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The sila-library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the sila-library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"path/filepath"

	"silachain/ethdb"
)

// The list of table names of chain freezer.
const (
	// ChainFreezerHeaderTable indicates the name of the freezer header table.
	ChainFreezerHeaderTable = "headers"

	// ChainFreezerHashTable indicates the name of the freezer canonical hash table.
	ChainFreezerHashTable = "hashes"

	// ChainFreezerBodiesTable indicates the name of the freezer block body table.
	ChainFreezerBodiesTable = "bodies"

	// ChainFreezerReceiptTable indicates the name of the freezer receipts table.
	ChainFreezerReceiptTable = "receipts"
)

// chainFreezerTableConfigs configures the settings for tables in the chain freezer.
var chainFreezerTableConfigs = map[string]freezerTableConfig{
	ChainFreezerHeaderTable:  {noSnappy: false, prunable: false},
	ChainFreezerHashTable:    {noSnappy: true, prunable: false},
	ChainFreezerBodiesTable:  {noSnappy: false, prunable: true},
	ChainFreezerReceiptTable: {noSnappy: false, prunable: true},
}

// freezerTableConfig contains the settings for a freezer table.
type freezerTableConfig struct {
	noSnappy bool
	prunable bool
}

const (
	// stateHistoryTableSize defines the maximum size of freezer data files.
	stateHistoryTableSize = 2 * 1000 * 1000 * 1000

	// stateHistoryAccountIndex indicates the name of the freezer state history table.
	stateHistoryMeta         = "history.meta"
	stateHistoryAccountIndex = "account.index"
	stateHistoryStorageIndex = "storage.index"
	stateHistoryAccountData  = "account.data"
	stateHistoryStorageData  = "storage.data"
)

// stateFreezerTableConfigs configures the settings for tables in the state freezer.
var stateFreezerTableConfigs = map[string]freezerTableConfig{
	stateHistoryMeta:         {noSnappy: true, prunable: true},
	stateHistoryAccountIndex: {noSnappy: false, prunable: true},
	stateHistoryStorageIndex: {noSnappy: false, prunable: true},
	stateHistoryAccountData:  {noSnappy: false, prunable: true},
	stateHistoryStorageData:  {noSnappy: false, prunable: true},
}

const (
	trienodeHistoryHeaderTable       = "trienode.header"
	trienodeHistoryKeySectionTable   = "trienode.key"
	trienodeHistoryValueSectionTable = "trienode.value"
)

// trienodeFreezerTableConfigs configures the settings for tables in the trienode freezer.
var trienodeFreezerTableConfigs = map[string]freezerTableConfig{
	trienodeHistoryHeaderTable:       {noSnappy: false, prunable: true},
	trienodeHistoryKeySectionTable:   {noSnappy: true, prunable: true},
	trienodeHistoryValueSectionTable: {noSnappy: true, prunable: true},
}

// The list of identifiers of ancient stores.
var (
	ChainFreezerName          = "chain"    // the folder name of chain segment ancient store.
	MerkleStateFreezerName    = "state"    // the folder name of state history ancient store.
	MerkleTrienodeFreezerName = "trienode" // the folder name of trienode history ancient store.
)

// freezers the collections of all builtin freezers.
var freezers = []string{
	ChainFreezerName,
	MerkleStateFreezerName,
	MerkleTrienodeFreezerName,
}

// NewStateFreezer initializes the ancient store for state history.
func NewStateFreezer(ancientDir string, readOnly bool) (ethdb.ResettableAncientStore, error) {
	if ancientDir == "" {
		return NewMemoryFreezer(readOnly, stateFreezerTableConfigs), nil
	}
	name := filepath.Join(ancientDir, MerkleStateFreezerName)
	return newResettableFreezer(name, "sila/db/state", readOnly, stateHistoryTableSize, stateFreezerTableConfigs)
}

// NewTrienodeFreezer initializes the ancient store for trienode history.
func NewTrienodeFreezer(ancientDir string, readOnly bool) (ethdb.ResettableAncientStore, error) {
	if ancientDir == "" {
		return NewMemoryFreezer(readOnly, trienodeFreezerTableConfigs), nil
	}
	name := filepath.Join(ancientDir, MerkleTrienodeFreezerName)
	return newResettableFreezer(name, "sila/db/trienode", readOnly, stateHistoryTableSize, trienodeFreezerTableConfigs)
}
