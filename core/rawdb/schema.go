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

// Package rawdb contains a collection of low level database accessors.
package rawdb

import (
	"bytes"
	"encoding/binary"

	"silachain/common"
	"silachain/crypto"
	"silachain/metrics"
)

// The fields below define the low level database schema prefixing.
var (
	// databaseVersionKey tracks the current database version.
	databaseVersionKey = []byte("DatabaseVersion")

	// headHeaderKey tracks the latest known header's hash.
	headHeaderKey = []byte("LastHeader")

	// headBlockKey tracks the latest known full block's hash.
	headBlockKey = []byte("LastBlock")

	// headFastBlockKey tracks the latest known incomplete block's hash during fast sync.
	headFastBlockKey = []byte("LastFast")

	// headFinalizedBlockKey tracks the latest known finalized block hash.
	headFinalizedBlockKey = []byte("LastFinalized")

	// persistentStateIDKey tracks the id of latest stored state(for path-based only).
	persistentStateIDKey = []byte("LastStateID")

	// lastPivotKey tracks the last pivot block used by fast sync (to reenable on sethead).
	lastPivotKey = []byte("LastPivot")

	// fastTrieProgressKey tracks the number of trie entries imported during fast sync.
	fastTrieProgressKey = []byte("TrieSync")

	// snapshotDisabledKey flags that the snapshot should not be maintained due to initial sync.
	snapshotDisabledKey = []byte("SnapshotDisabled")

	// SnapshotRootKey tracks the hash of the last snapshot.
	SnapshotRootKey = []byte("SnapshotRoot")

	// snapshotJournalKey tracks the in-memory diff layers across restarts.
	snapshotJournalKey = []byte("SnapshotJournal")

	// snapshotGeneratorKey tracks the snapshot generation marker across restarts.
	snapshotGeneratorKey = []byte("SnapshotGenerator")

	// snapshotRecoveryKey tracks the snapshot recovery marker across restarts.
	snapshotRecoveryKey = []byte("SnapshotRecovery")

	// snapshotSyncStatusKey tracks the snapshot sync status across restarts.
	snapshotSyncStatusKey = []byte("SnapshotSyncStatus")

	// skeletonSyncStatusKey tracks the skeleton sync status across restarts.
	skeletonSyncStatusKey = []byte("SkeletonSyncStatus")

	// trieJournalKey tracks the in-memory trie node layers across restarts.
	trieJournalKey = []byte("TrieJournal")

	// headStateHistoryIndexKey tracks the ID of the latest state history that has been indexed.
	headStateHistoryIndexKey = []byte("LastStateHistoryIndex")

	// headTrienodeHistoryIndexKey tracks the ID of the latest state history that has been indexed.
	headTrienodeHistoryIndexKey = []byte("LastTrienodeHistoryIndex")

	// txIndexTailKey tracks the oldest block whose transactions have been indexed.
	txIndexTailKey = []byte("TransactionIndexTail")

	// fastTxLookupLimitKey tracks the transaction lookup limit during fast sync (deprecated).
	fastTxLookupLimitKey = []byte("FastTransactionLookupLimit")

	// badBlockKey tracks the list of bad blocks seen by local.
	badBlockKey = []byte("InvalidBlock")

	// uncleanShutdownKey tracks the list of local crashes.
	uncleanShutdownKey = []byte("unclean-shutdown")

	// transitionStatusKey tracks the transition status (deprecated).
	transitionStatusKey = []byte("transition")

	// snapSyncStatusFlagKey flags that status of snap sync.
	snapSyncStatusFlagKey = []byte("SnapSyncStatus")

	// Data item prefixes.
	headerPrefix       = []byte("h")
	headerTDSuffix     = []byte("t")
	headerHashSuffix   = []byte("n")
	headerNumberPrefix = []byte("H")

	blockBodyPrefix     = []byte("b")
	blockReceiptsPrefix = []byte("r")
	accessListPrefix    = []byte("j")

	txLookupPrefix        = []byte("l")
	bloomBitsPrefix       = []byte("B")
	SnapshotAccountPrefix = []byte("a")
	SnapshotStoragePrefix = []byte("o")
	CodePrefix            = []byte("c")
	skeletonHeaderPrefix  = []byte("S")

	// Path-based storage scheme of merkle patricia trie.
	TrieNodeAccountPrefix = []byte("A")
	TrieNodeStoragePrefix = []byte("O")
	stateIDPrefix         = []byte("L")

	// State history indexing within path-based storage scheme.
	StateHistoryIndexPrefix           = []byte("m")
	StateHistoryAccountMetadataPrefix = []byte("ma")
	StateHistoryStorageMetadataPrefix = []byte("ms")
	TrienodeHistoryMetadataPrefix     = []byte("mt")
	StateHistoryAccountBlockPrefix    = []byte("mba")
	StateHistoryStorageBlockPrefix    = []byte("mbs")
	TrienodeHistoryBlockPrefix        = []byte("mbt")

	// VerklePrefix is the database prefix for Verkle trie data.
	VerklePrefix = []byte("v")

	PreimagePrefix = []byte("secure-key-")
	configPrefix   = []byte("sila-config-")
	genesisPrefix  = []byte("sila-genesis-")

	CliqueSnapshotPrefix = []byte("clique-")

	BestUpdateKey         = []byte("update-")
	FixedCommitteeRootKey = []byte("fixedRoot-")
	SyncCommitteeKey      = []byte("committee-")

	// new log index
	filterMapsPrefix         = "fm-"
	filterMapsRangeKey       = []byte(filterMapsPrefix + "R")
	filterMapRowPrefix       = []byte(filterMapsPrefix + "r")
	filterMapLastBlockPrefix = []byte(filterMapsPrefix + "b")
	filterMapBlockLVPrefix   = []byte(filterMapsPrefix + "p")

	// old log index
	bloomBitsMetaPrefix = []byte("iB")

	preimageCounter     = metrics.NewRegisteredCounter("db/preimage/total", nil)
	preimageHitsCounter = metrics.NewRegisteredCounter("db/preimage/hits", nil)
	preimageMissCounter = metrics.NewRegisteredCounter("db/preimage/miss", nil)

	// Overlay transition information
	OverlayTransitionStatePrefix = []byte("overlay-transition-state-")
)

// LegacyTxLookupEntry is the legacy TxLookupEntry definition.
type LegacyTxLookupEntry struct {
	BlockHash  common.Hash
	BlockIndex uint64
	Index      uint64
}

// encodeBlockNumber encodes a block number as big endian uint64.
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// headerKeyPrefix = headerPrefix + num (uint64 big endian).
func headerKeyPrefix(number uint64) []byte {
	return append(headerPrefix, encodeBlockNumber(number)...)
}

// headerKey = headerPrefix + num (uint64 big endian) + hash.
func headerKey(number uint64, hash common.Hash) []byte {
	return append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// headerHashKey = headerPrefix + num (uint64 big endian) + headerHashSuffix.
func headerHashKey(number uint64) []byte {
	return append(append(headerPrefix, encodeBlockNumber(number)...), headerHashSuffix...)
}

// headerNumberKey = headerNumberPrefix + hash.
func headerNumberKey(hash common.Hash) []byte {
	return append(headerNumberPrefix, hash.Bytes()...)
}

// blockBodyKey = blockBodyPrefix + num (uint64 big endian) + hash.
func blockBodyKey(number uint64, hash common.Hash) []byte {
	return append(append(blockBodyPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// blockReceiptsKey = blockReceiptsPrefix + num (uint64 big endian) + hash.
func blockReceiptsKey(number uint64, hash common.Hash) []byte {
	return append(append(blockReceiptsPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// accessListKey = accessListPrefix + num (uint64 big endian) + hash.
func accessListKey(number uint64, hash common.Hash) []byte {
	return append(append(accessListPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// txLookupKey = txLookupPrefix + hash.
func txLookupKey(hash common.Hash) []byte {
	return append(txLookupPrefix, hash.Bytes()...)
}

// accountSnapshotKey = SnapshotAccountPrefix + hash.
func accountSnapshotKey(hash common.Hash) []byte {
	return append(SnapshotAccountPrefix, hash.Bytes()...)
}

// storageSnapshotKey = SnapshotStoragePrefix + account hash + storage hash.
func storageSnapshotKey(accountHash, storageHash common.Hash) []byte {
	buf := make([]byte, len(SnapshotStoragePrefix)+common.HashLength+common.HashLength)
	n := copy(buf, SnapshotStoragePrefix)
	n += copy(buf[n:], accountHash.Bytes())
	copy(buf[n:], storageHash.Bytes())
	return buf
}

// storageSnapshotsKey = SnapshotStoragePrefix + account hash.
func storageSnapshotsKey(accountHash common.Hash) []byte {
	return append(SnapshotStoragePrefix, accountHash.Bytes()...)
}

// skeletonHeaderKey = skeletonHeaderPrefix + num (uint64 big endian).
func skeletonHeaderKey(number uint64) []byte {
	return append(skeletonHeaderPrefix, encodeBlockNumber(number)...)
}

// preimageKey = PreimagePrefix + hash.
func preimageKey(hash common.Hash) []byte {
	return append(PreimagePrefix, hash.Bytes()...)
}

// codeKey = CodePrefix + hash.
func codeKey(hash common.Hash) []byte {
	return append(CodePrefix, hash.Bytes()...)
}

// IsCodeKey reports whether the given byte slice is the key of contract code.
func IsCodeKey(key []byte) (bool, []byte) {
	if bytes.HasPrefix(key, CodePrefix) && len(key) == common.HashLength+len(CodePrefix) {
		return true, key[len(CodePrefix):]
	}
	return false, nil
}

// configKey = configPrefix + hash.
func configKey(hash common.Hash) []byte {
	return append(configPrefix, hash.Bytes()...)
}

// genesisStateSpecKey = genesisPrefix + hash.
func genesisStateSpecKey(hash common.Hash) []byte {
	return append(genesisPrefix, hash.Bytes()...)
}

// stateIDKey = stateIDPrefix + root (32 bytes).
func stateIDKey(root common.Hash) []byte {
	return append(stateIDPrefix, root.Bytes()...)
}

// accountTrieNodeKey = TrieNodeAccountPrefix + nodePath.
func accountTrieNodeKey(path []byte) []byte {
	return append(TrieNodeAccountPrefix, path...)
}

// storageTrieNodeKey = TrieNodeStoragePrefix + accountHash + nodePath.
func storageTrieNodeKey(accountHash common.Hash, path []byte) []byte {
	buf := make([]byte, len(TrieNodeStoragePrefix)+common.HashLength+len(path))
	n := copy(buf, TrieNodeStoragePrefix)
	n += copy(buf[n:], accountHash.Bytes())
	copy(buf[n:], path)
	return buf
}

// IsLegacyTrieNode reports whether a provided database entry is a legacy trie node.
func IsLegacyTrieNode(key []byte, val []byte) bool {
	if len(key) != common.HashLength {
		return false
	}
	return bytes.Equal(key, crypto.Keccak256(val))
}

// ResolveAccountTrieNodeKey reports whether a provided database entry is an account trie node.
func ResolveAccountTrieNodeKey(key []byte) (bool, []byte) {
	if !bytes.HasPrefix(key, TrieNodeAccountPrefix) {
		return false, nil
	}
	if len(key) >= len(TrieNodeAccountPrefix)+common.HashLength*2 {
		return false, nil
	}
	return true, key[len(TrieNodeAccountPrefix):]
}

// IsAccountTrieNode reports whether a provided database entry is an account trie node.
func IsAccountTrieNode(key []byte) bool {
	ok, _ := ResolveAccountTrieNodeKey(key)
	return ok
}

// ResolveStorageTrieNode reports whether a provided database entry is a storage trie node.
func ResolveStorageTrieNode(key []byte) (bool, common.Hash, []byte) {
	if !bytes.HasPrefix(key, TrieNodeStoragePrefix) {
		return false, common.Hash{}, nil
	}
	if len(key) < len(TrieNodeStoragePrefix)+common.HashLength {
		return false, common.Hash{}, nil
	}
	if len(key) >= len(TrieNodeStoragePrefix)+common.HashLength+common.HashLength*2 {
		return false, common.Hash{}, nil
	}
	accountHash := common.BytesToHash(key[len(TrieNodeStoragePrefix) : len(TrieNodeStoragePrefix)+common.HashLength])
	return true, accountHash, key[len(TrieNodeStoragePrefix)+common.HashLength:]
}

// IsStorageTrieNode reports whether a provided database entry is a storage trie node.
func IsStorageTrieNode(key []byte) bool {
	ok, _, _ := ResolveStorageTrieNode(key)
	return ok
}

// filterMapRowKey = filterMapRowPrefix + mapRowIndex (uint64 big endian).
func filterMapRowKey(mapRowIndex uint64, base bool) []byte {
	extLen := 8
	if base {
		extLen = 9
	}
	l := len(filterMapRowPrefix)
	key := make([]byte, l+extLen)
	copy(key[:l], filterMapRowPrefix)
	binary.BigEndian.PutUint64(key[l:l+8], mapRowIndex)
	return key
}

// filterMapLastBlockKey = filterMapLastBlockPrefix + mapIndex (uint32 big endian).
func filterMapLastBlockKey(mapIndex uint32) []byte {
	l := len(filterMapLastBlockPrefix)
	key := make([]byte, l+4)
	copy(key[:l], filterMapLastBlockPrefix)
	binary.BigEndian.PutUint32(key[l:], mapIndex)
	return key
}

// filterMapBlockLVKey = filterMapBlockLVPrefix + num (uint64 big endian).
func filterMapBlockLVKey(number uint64) []byte {
	l := len(filterMapBlockLVPrefix)
	key := make([]byte, l+8)
	copy(key[:l], filterMapBlockLVPrefix)
	binary.BigEndian.PutUint64(key[l:], number)
	return key
}

// accountHistoryIndexKey = StateHistoryAccountMetadataPrefix + addressHash.
func accountHistoryIndexKey(addressHash common.Hash) []byte {
	return append(StateHistoryAccountMetadataPrefix, addressHash.Bytes()...)
}

// storageHistoryIndexKey = StateHistoryStorageMetadataPrefix + addressHash + storageHash.
func storageHistoryIndexKey(addressHash common.Hash, storageHash common.Hash) []byte {
	totalLen := len(StateHistoryStorageMetadataPrefix) + 2*common.HashLength
	out := make([]byte, totalLen)

	off := 0
	off += copy(out[off:], StateHistoryStorageMetadataPrefix)
	off += copy(out[off:], addressHash.Bytes())
	copy(out[off:], storageHash.Bytes())

	return out
}

// trienodeHistoryIndexKey = TrienodeHistoryMetadataPrefix + addressHash + trienode path.
func trienodeHistoryIndexKey(addressHash common.Hash, path []byte) []byte {
	totalLen := len(TrienodeHistoryMetadataPrefix) + common.HashLength + len(path)
	out := make([]byte, totalLen)

	off := 0
	off += copy(out[off:], TrienodeHistoryMetadataPrefix)
	off += copy(out[off:], addressHash.Bytes())
	copy(out[off:], path)

	return out
}

// accountHistoryIndexBlockKey = StateHistoryAccountBlockPrefix + addressHash + blockID.
func accountHistoryIndexBlockKey(addressHash common.Hash, blockID uint32) []byte {
	totalLen := len(StateHistoryAccountBlockPrefix) + common.HashLength + 4
	out := make([]byte, totalLen)

	off := 0
	off += copy(out[off:], StateHistoryAccountBlockPrefix)
	off += copy(out[off:], addressHash.Bytes())
	binary.BigEndian.PutUint32(out[off:], blockID)

	return out
}

// storageHistoryIndexBlockKey = StateHistoryStorageBlockPrefix + addressHash + storageHash + blockID.
func storageHistoryIndexBlockKey(addressHash common.Hash, storageHash common.Hash, blockID uint32) []byte {
	totalLen := len(StateHistoryStorageBlockPrefix) + 2*common.HashLength + 4
	out := make([]byte, totalLen)

	off := 0
	off += copy(out[off:], StateHistoryStorageBlockPrefix)
	off += copy(out[off:], addressHash.Bytes())
	off += copy(out[off:], storageHash.Bytes())
	binary.BigEndian.PutUint32(out[off:], blockID)

	return out
}

// trienodeHistoryIndexBlockKey = TrienodeHistoryBlockPrefix + addressHash + trienode path + blockID.
func trienodeHistoryIndexBlockKey(addressHash common.Hash, path []byte, blockID uint32) []byte {
	totalLen := len(TrienodeHistoryBlockPrefix) + common.HashLength + len(path) + 4
	out := make([]byte, totalLen)

	off := 0
	off += copy(out[off:], TrienodeHistoryBlockPrefix)
	off += copy(out[off:], addressHash.Bytes())
	off += copy(out[off:], path)
	binary.BigEndian.PutUint32(out[off:], blockID)

	return out
}

// transitionStateKey = transitionStatusKey + hash.
func transitionStateKey(hash common.Hash) []byte {
	return append(OverlayTransitionStatePrefix, hash.Bytes()...)
}
