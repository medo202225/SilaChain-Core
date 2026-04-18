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

package snapshot

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/holiman/uint256"
	"silachain/common"
	"silachain/core/rawdb"
	"silachain/core/types"
	"silachain/crypto"
	"silachain/ethdb"
	"silachain/log"
	"silachain/rlp"
	"silachain/trie"
	"silachain/trie/trienode"
	"silachain/triedb"
	"silachain/triedb/hashdb"
	"silachain/triedb/pathdb"
)

func hashData(input []byte) common.Hash {
	return crypto.Keccak256Hash(input)
}

// Tests that snapshot generation from an empty database.
func TestGeneration(t *testing.T) {
	testGeneration(t, rawdb.HashScheme)
	testGeneration(t, rawdb.PathScheme)
}

func testGeneration(t *testing.T, scheme string) {
	var helper = newHelper(scheme)
	stRoot := helper.makeStorageTrie("", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, false)

	helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})

	helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)

	root, snap := helper.CommitAndGenerate()
	if have, want := root, common.HexToHash("0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd"); have != want {
		t.Fatalf("have %#x want %#x", have, want)
	}
	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)

	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation with existent flat state.
func TestGenerateExistentState(t *testing.T) {
	testGenerateExistentState(t, rawdb.HashScheme)
	testGenerateExistentState(t, rawdb.PathScheme)
}

func testGenerateExistentState(t *testing.T, scheme string) {
	var helper = newHelper(scheme)

	stRoot := helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addSnapAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addSnapStorage("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addSnapAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})

	stRoot = helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addSnapAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addSnapStorage("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	root, snap := helper.CommitAndGenerate()
	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)

	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

func checkSnapRoot(t *testing.T, snap *diskLayer, trieRoot common.Hash) {
	t.Helper()

	accIt := snap.AccountIterator(common.Hash{})
	defer accIt.Release()

	snapRoot, err := generateTrieRoot(nil, "", accIt, common.Hash{}, stackTrieGenerate,
		func(db ethdb.KeyValueWriter, accountHash, codeHash common.Hash, stat *generateStats) (common.Hash, error) {
			storageIt := snap.StorageIterator(accountHash, common.Hash{})
			defer storageIt.Release()

			hash, err := generateTrieRoot(nil, "", storageIt, accountHash, stackTrieGenerate, nil, stat, false)
			if err != nil {
				return common.Hash{}, err
			}
			return hash, nil
		}, newGenerateStats(), true)
	if err != nil {
		t.Fatal(err)
	}
	if snapRoot != trieRoot {
		t.Fatalf("snaproot: %#x != trieroot #%x", snapRoot, trieRoot)
	}
	if err := CheckDanglingStorage(snap.diskdb); err != nil {
		t.Fatalf("Detected dangling storages: %v", err)
	}
}

type testHelper struct {
	diskdb  ethdb.Database
	triedb  *triedb.Database
	accTrie *trie.StateTrie
	nodes   *trienode.MergedNodeSet
	states  *triedb.StateSet
}

func newHelper(scheme string) *testHelper {
	diskdb := rawdb.NewMemoryDatabase()
	config := &triedb.Config{}
	if scheme == rawdb.PathScheme {
		config.PathDB = &pathdb.Config{
			SnapshotNoBuild: true,
			NoAsyncFlush:    true,
		}
	} else {
		config.HashDB = &hashdb.Config{}
	}
	db := triedb.NewDatabase(diskdb, config)
	accTrie, _ := trie.NewStateTrie(trie.StateTrieID(types.EmptyRootHash), db)
	return &testHelper{
		diskdb:  diskdb,
		triedb:  db,
		accTrie: accTrie,
		nodes:   trienode.NewMergedNodeSet(),
		states:  triedb.NewStateSet(),
	}
}

func (t *testHelper) addTrieAccount(acckey string, acc *types.StateAccount) {
	val, _ := rlp.EncodeToBytes(acc)
	t.accTrie.MustUpdate([]byte(acckey), val)

	accHash := hashData([]byte(acckey))
	t.states.Accounts[accHash] = val
	t.states.AccountsOrigin[common.BytesToAddress([]byte(acckey))] = nil
}

func (t *testHelper) addSnapAccount(acckey string, acc *types.StateAccount) {
	key := hashData([]byte(acckey))
	rawdb.WriteAccountSnapshot(t.diskdb, key, types.SlimAccountRLP(*acc))
}

func (t *testHelper) addAccount(acckey string, acc *types.StateAccount) {
	t.addTrieAccount(acckey, acc)
	t.addSnapAccount(acckey, acc)
}

func (t *testHelper) addSnapStorage(accKey string, keys []string, vals []string) {
	accHash := hashData([]byte(accKey))
	for i, key := range keys {
		rawdb.WriteStorageSnapshot(t.diskdb, accHash, hashData([]byte(key)), []byte(vals[i]))
	}
}

func (t *testHelper) makeStorageTrie(accKey string, keys []string, vals []string, commit bool) common.Hash {
	owner := hashData([]byte(accKey))
	addr := common.BytesToAddress([]byte(accKey))
	id := trie.StorageTrieID(types.EmptyRootHash, owner, types.EmptyRootHash)
	stTrie, _ := trie.NewStateTrie(id, t.triedb)
	for i, k := range keys {
		stTrie.MustUpdate([]byte(k), []byte(vals[i]))
		if t.states.Storages[owner] == nil {
			t.states.Storages[owner] = make(map[common.Hash][]byte)
		}
		if t.states.StoragesOrigin[addr] == nil {
			t.states.StoragesOrigin[addr] = make(map[common.Hash][]byte)
		}
		t.states.Storages[owner][hashData([]byte(k))] = []byte(vals[i])
		t.states.StoragesOrigin[addr][hashData([]byte(k))] = nil
	}
	if !commit {
		return stTrie.Hash()
	}
	root, nodes := stTrie.Commit(false)
	if nodes != nil {
		t.nodes.Merge(nodes)
	}
	return root
}

func (t *testHelper) Commit() common.Hash {
	root, nodes := t.accTrie.Commit(true)
	if nodes != nil {
		t.nodes.Merge(nodes)
	}
	t.triedb.Update(root, types.EmptyRootHash, 0, t.nodes, t.states)
	t.triedb.Commit(root, false)
	return root
}

func (t *testHelper) CommitAndGenerate() (common.Hash, *diskLayer) {
	root := t.Commit()
	snap := generateSnapshot(t.diskdb, t.triedb, 16, root)
	return root, snap
}

// Tests that snapshot generation with existent flat state, where the flat state contains some errors.
func TestGenerateExistentStateWithWrongStorage(t *testing.T) {
	testGenerateExistentStateWithWrongStorage(t, rawdb.HashScheme)
	testGenerateExistentStateWithWrongStorage(t, rawdb.PathScheme)
}

func testGenerateExistentStateWithWrongStorage(t *testing.T, scheme string) {
	helper := newHelper(scheme)

	helper.addAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addSnapStorage("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	stRoot := helper.makeStorageTrie("acc-2", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})

	{
		helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-3", []string{"key-2", "key-3"}, []string{"val-2", "val-3"})

		helper.makeStorageTrie("acc-4", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-4", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-4", []string{"key-1", "key-3"}, []string{"val-1", "val-3"})

		helper.makeStorageTrie("acc-5", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-5", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-5", []string{"key-1", "key-2"}, []string{"val-1", "val-2"})
	}

	{
		helper.makeStorageTrie("acc-6", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-6", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-6", []string{"key-1", "key-2", "key-3"}, []string{"badval-1", "val-2", "val-3"})

		helper.makeStorageTrie("acc-7", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-7", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-7", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "badval-2", "val-3"})

		helper.makeStorageTrie("acc-8", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-8", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-8", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "badval-3"})

		helper.makeStorageTrie("acc-9", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-9", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-9", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-3", "val-2"})
	}

	{
		helper.makeStorageTrie("acc-10", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-10", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-10", []string{"key-0", "key-1", "key-2", "key-3"}, []string{"val-0", "val-1", "val-2", "val-3"})

		helper.makeStorageTrie("acc-11", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-11", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-11", []string{"key-1", "key-2", "key-2-1", "key-3"}, []string{"val-1", "val-2", "val-2-1", "val-3"})

		helper.makeStorageTrie("acc-12", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-12", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-12", []string{"key-1", "key-2", "key-3", "key-4"}, []string{"val-1", "val-2", "val-3", "val-4"})
	}

	root, snap := helper.CommitAndGenerate()
	t.Logf("Root: %#x\n", root)

	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation with existent flat state, where the flat state contains wrong accounts.
func TestGenerateExistentStateWithWrongAccounts(t *testing.T) {
	testGenerateExistentStateWithWrongAccounts(t, rawdb.HashScheme)
	testGenerateExistentStateWithWrongAccounts(t, rawdb.PathScheme)
}

func testGenerateExistentStateWithWrongAccounts(t *testing.T, scheme string) {
	helper := newHelper(scheme)

	helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.makeStorageTrie("acc-2", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.makeStorageTrie("acc-4", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	stRoot := helper.makeStorageTrie("acc-6", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)

	{
		helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addTrieAccount("acc-4", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addTrieAccount("acc-6", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	}

	{
		helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: common.Hex2Bytes("0x1234")})

		helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	}

	{
		helper.addSnapAccount("acc-0", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapAccount("acc-5", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: common.Hex2Bytes("0x1234")})
		helper.addSnapAccount("acc-7", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	}

	root, snap := helper.CommitAndGenerate()
	t.Logf("Root: %#x\n", root)

	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)

	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation errors out correctly in case of a missing trie node.
func TestGenerateCorruptAccountTrie(t *testing.T) {
	testGenerateCorruptAccountTrie(t, rawdb.HashScheme)
	testGenerateCorruptAccountTrie(t, rawdb.PathScheme)
}

func testGenerateCorruptAccountTrie(t *testing.T, scheme string) {
	helper := newHelper(scheme)

	helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})

	root := helper.Commit()

	targetPath := []byte{0xc}
	targetHash := common.HexToHash("0x65145f923027566669a1ae5ccac66f945b55ff6eaeb17d2ea8e048b7d381f2d7")

	rawdb.DeleteTrieNode(helper.diskdb, common.Hash{}, targetPath, targetHash, scheme)

	snap := generateSnapshot(helper.diskdb, helper.triedb, 16, root)
	select {
	case <-snap.genPending:
		t.Errorf("Snapshot generated against corrupt account trie")
	case <-time.After(time.Second):
	}
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation errors out correctly in case of a missing root trie node.
func TestGenerateMissingStorageTrie(t *testing.T) {
	testGenerateMissingStorageTrie(t, rawdb.HashScheme)
	testGenerateMissingStorageTrie(t, rawdb.PathScheme)
}

func testGenerateMissingStorageTrie(t *testing.T, scheme string) {
	var (
		acc1   = hashData([]byte("acc-1"))
		acc3   = hashData([]byte("acc-3"))
		helper = newHelper(scheme)
	)
	stRoot := helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	stRoot = helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})

	root := helper.Commit()

	rawdb.DeleteTrieNode(helper.diskdb, acc1, nil, stRoot, scheme)
	rawdb.DeleteTrieNode(helper.diskdb, acc3, nil, stRoot, scheme)

	snap := generateSnapshot(helper.diskdb, helper.triedb, 16, root)
	select {
	case <-snap.genPending:
		t.Errorf("Snapshot generated against corrupt storage trie")
	case <-time.After(time.Second):
	}
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation errors out correctly in case of a missing trie node in a storage trie.
func TestGenerateCorruptStorageTrie(t *testing.T) {
	testGenerateCorruptStorageTrie(t, rawdb.HashScheme)
	testGenerateCorruptStorageTrie(t, rawdb.PathScheme)
}

func testGenerateCorruptStorageTrie(t *testing.T, scheme string) {
	helper := newHelper(scheme)

	stRoot := helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	stRoot = helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})

	root := helper.Commit()

	targetPath := []byte{0x4}
	targetHash := common.HexToHash("0x18a0f4d79cff4459642dd7604f303886ad9d77c30cf3d7d7cedb3a693ab6d371")
	rawdb.DeleteTrieNode(helper.diskdb, hashData([]byte("acc-1")), targetPath, targetHash, scheme)
	rawdb.DeleteTrieNode(helper.diskdb, hashData([]byte("acc-3")), targetPath, targetHash, scheme)

	snap := generateSnapshot(helper.diskdb, helper.triedb, 16, root)
	select {
	case <-snap.genPending:
		t.Errorf("Snapshot generated against corrupt storage trie")
	case <-time.After(time.Second):
	}
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation when an extra account with storage exists.
func TestGenerateWithExtraAccounts(t *testing.T) {
	testGenerateWithExtraAccounts(t, rawdb.HashScheme)
	testGenerateWithExtraAccounts(t, rawdb.PathScheme)
}

func testGenerateWithExtraAccounts(t *testing.T, scheme string) {
	helper := newHelper(scheme)
	{
		stRoot := helper.makeStorageTrie("acc-1",
			[]string{"key-1", "key-2", "key-3", "key-4", "key-5"},
			[]string{"val-1", "val-2", "val-3", "val-4", "val-5"},
			true,
		)
		acc := &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()}
		val, _ := rlp.EncodeToBytes(acc)
		helper.accTrie.MustUpdate([]byte("acc-1"), val)

		key := hashData([]byte("acc-1"))
		rawdb.WriteAccountSnapshot(helper.diskdb, key, val)
		rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-1")), []byte("val-1"))
		rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-2")), []byte("val-2"))
		rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-3")), []byte("val-3"))
		rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-4")), []byte("val-4"))
		rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-5")), []byte("val-5"))
	}
	{
		stRoot := helper.makeStorageTrie("acc-2",
			[]string{"key-1", "key-2", "key-3", "key-4", "key-5"},
			[]string{"val-1", "val-2", "val-3", "val-4", "val-5"},
			true,
		)
		acc := &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()}
		val, _ := rlp.EncodeToBytes(acc)
		key := hashData([]byte("acc-2"))
		rawdb.WriteAccountSnapshot(helper.diskdb, key, val)
		rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("b-key-1")), []byte("b-val-1"))
		rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("b-key-2")), []byte("b-val-2"))
		rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("b-key-3")), []byte("b-val-3"))
	}
	root := helper.Commit()

	if data := rawdb.ReadStorageSnapshot(helper.diskdb, hashData([]byte("acc-2")), hashData([]byte("b-key-1"))); data == nil {
		t.Fatalf("expected snap storage to exist")
	}
	snap := generateSnapshot(helper.diskdb, helper.triedb, 16, root)
	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)

	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
	if data := rawdb.ReadStorageSnapshot(helper.diskdb, hashData([]byte("acc-2")), hashData([]byte("b-key-1"))); data != nil {
		t.Fatalf("expected slot to be removed, got %v", string(data))
	}
}

func enableLogging() {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelTrace, true)))
}

// Tests that snapshot generation when an extra account with storage exists.
func TestGenerateWithManyExtraAccounts(t *testing.T) {
	testGenerateWithManyExtraAccounts(t, rawdb.HashScheme)
	testGenerateWithManyExtraAccounts(t, rawdb.PathScheme)
}

func testGenerateWithManyExtraAccounts(t *testing.T, scheme string) {
	if false {
		enableLogging()
	}
	helper := newHelper(scheme)
	{
		stRoot := helper.makeStorageTrie("acc-1",
			[]string{"key-1", "key-2", "key-3"},
			[]string{"val-1", "val-2", "val-3"},
			true,
		)
		acc := &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()}
		val, _ := rlp.EncodeToBytes(acc)
		helper.accTrie.MustUpdate([]byte("acc-1"), val)

		key := hashData([]byte("acc-1"))
		rawdb.WriteAccountSnapshot(helper.diskdb, key, val)
		rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-1")), []byte("val-1"))
		rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-2")), []byte("val-2"))
		rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-3")), []byte("val-3"))
	}
	{
		for i := 0; i < 1000; i++ {
			acc := &types.StateAccount{Balance: uint256.NewInt(uint64(i)), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}
			val, _ := rlp.EncodeToBytes(acc)
			key := hashData(fmt.Appendf(nil, "acc-%d", i))
			rawdb.WriteAccountSnapshot(helper.diskdb, key, val)
		}
	}
	root, snap := helper.CommitAndGenerate()
	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests snapshot generation with extra accounts before and after.
func TestGenerateWithExtraBeforeAndAfter(t *testing.T) {
	testGenerateWithExtraBeforeAndAfter(t, rawdb.HashScheme)
	testGenerateWithExtraBeforeAndAfter(t, rawdb.PathScheme)
}

func testGenerateWithExtraBeforeAndAfter(t *testing.T, scheme string) {
	accountCheckRange = 3
	if false {
		enableLogging()
	}
	helper := newHelper(scheme)
	{
		acc := &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}
		val, _ := rlp.EncodeToBytes(acc)
		helper.accTrie.MustUpdate(common.HexToHash("0x03").Bytes(), val)
		helper.accTrie.MustUpdate(common.HexToHash("0x07").Bytes(), val)

		rawdb.WriteAccountSnapshot(helper.diskdb, common.HexToHash("0x01"), val)
		rawdb.WriteAccountSnapshot(helper.diskdb, common.HexToHash("0x02"), val)
		rawdb.WriteAccountSnapshot(helper.diskdb, common.HexToHash("0x03"), val)
		rawdb.WriteAccountSnapshot(helper.diskdb, common.HexToHash("0x04"), val)
		rawdb.WriteAccountSnapshot(helper.diskdb, common.HexToHash("0x05"), val)
		rawdb.WriteAccountSnapshot(helper.diskdb, common.HexToHash("0x06"), val)
		rawdb.WriteAccountSnapshot(helper.diskdb, common.HexToHash("0x07"), val)
	}
	root, snap := helper.CommitAndGenerate()
	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// TestGenerateWithMalformedSnapdata tests what happens if we have junk in the snapshot database.
func TestGenerateWithMalformedSnapdata(t *testing.T) {
	testGenerateWithMalformedSnapdata(t, rawdb.HashScheme)
	testGenerateWithMalformedSnapdata(t, rawdb.PathScheme)
}

func testGenerateWithMalformedSnapdata(t *testing.T, scheme string) {
	accountCheckRange = 3
	if false {
		enableLogging()
	}
	helper := newHelper(scheme)
	{
		acc := &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}
		val, _ := rlp.EncodeToBytes(acc)
		helper.accTrie.MustUpdate(common.HexToHash("0x03").Bytes(), val)

		junk := make([]byte, 100)
		copy(junk, []byte{0xde, 0xad})
		rawdb.WriteAccountSnapshot(helper.diskdb, common.HexToHash("0x02"), junk)
		rawdb.WriteAccountSnapshot(helper.diskdb, common.HexToHash("0x03"), junk)
		rawdb.WriteAccountSnapshot(helper.diskdb, common.HexToHash("0x04"), junk)
		rawdb.WriteAccountSnapshot(helper.diskdb, common.HexToHash("0x05"), junk)
	}
	root, snap := helper.CommitAndGenerate()
	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
	if data := rawdb.ReadStorageSnapshot(helper.diskdb, hashData([]byte("acc-2")), hashData([]byte("b-key-1"))); data != nil {
		t.Fatalf("expected slot to be removed, got %v", string(data))
	}
}

func TestGenerateFromEmptySnap(t *testing.T) {
	testGenerateFromEmptySnap(t, rawdb.HashScheme)
	testGenerateFromEmptySnap(t, rawdb.PathScheme)
}

func testGenerateFromEmptySnap(t *testing.T, scheme string) {
	accountCheckRange = 10
	storageCheckRange = 20
	helper := newHelper(scheme)
	for i := 0; i < 400; i++ {
		stRoot := helper.makeStorageTrie(fmt.Sprintf("acc-%d", i), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addTrieAccount(fmt.Sprintf("acc-%d", i),
			&types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	}
	root, snap := helper.CommitAndGenerate()
	t.Logf("Root: %#x\n", root)

	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation with incomplete storage.
func TestGenerateWithIncompleteStorage(t *testing.T) {
	testGenerateWithIncompleteStorage(t, rawdb.HashScheme)
	testGenerateWithIncompleteStorage(t, rawdb.PathScheme)
}

func testGenerateWithIncompleteStorage(t *testing.T, scheme string) {
	storageCheckRange = 4
	helper := newHelper(scheme)
	stKeys := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	stVals := []string{"v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8"}
	for i := 0; i < 8; i++ {
		accKey := fmt.Sprintf("acc-%d", i)
		stRoot := helper.makeStorageTrie(accKey, stKeys, stVals, true)
		helper.addAccount(accKey, &types.StateAccount{Balance: uint256.NewInt(uint64(i)), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		var moddedKeys []string
		var moddedVals []string
		for ii := 0; ii < 8; ii++ {
			if ii != i {
				moddedKeys = append(moddedKeys, stKeys[ii])
				moddedVals = append(moddedVals, stVals[ii])
			}
		}
		helper.addSnapStorage(accKey, moddedKeys, moddedVals)
	}
	root, snap := helper.CommitAndGenerate()
	t.Logf("Root: %#x\n", root)

	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

func incKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]++
		if key[i] != 0x0 {
			break
		}
	}
	return key
}

func decKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]--
		if key[i] != 0xff {
			break
		}
	}
	return key
}

func populateDangling(disk ethdb.KeyValueStore) {
	populate := func(accountHash common.Hash, keys []string, vals []string) {
		for i, key := range keys {
			rawdb.WriteStorageSnapshot(disk, accountHash, hashData([]byte(key)), []byte(vals[i]))
		}
	}
	populate(common.Hash{}, []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	populate(common.HexToHash("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	hash := decKey(hashData([]byte("acc-1")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	hash = incKey(hashData([]byte("acc-1")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	hash = decKey(hashData([]byte("acc-2")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	hash = incKey(hashData([]byte("acc-2")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	hash = decKey(hashData([]byte("acc-3")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	hash = incKey(hashData([]byte("acc-3")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	populate(randomHash(), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	populate(randomHash(), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	populate(randomHash(), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
}

// Tests that snapshot generation with dangling storages.
func TestGenerateCompleteSnapshotWithDanglingStorage(t *testing.T) {
	testGenerateCompleteSnapshotWithDanglingStorage(t, rawdb.HashScheme)
	testGenerateCompleteSnapshotWithDanglingStorage(t, rawdb.PathScheme)
}

func testGenerateCompleteSnapshotWithDanglingStorage(t *testing.T, scheme string) {
	var helper = newHelper(scheme)

	stRoot := helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})

	helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})

	helper.addSnapStorage("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	helper.addSnapStorage("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	populateDangling(helper.diskdb)

	root, snap := helper.CommitAndGenerate()
	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)

	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation with broken dangling storages.
func TestGenerateBrokenSnapshotWithDanglingStorage(t *testing.T) {
	testGenerateBrokenSnapshotWithDanglingStorage(t, rawdb.HashScheme)
	testGenerateBrokenSnapshotWithDanglingStorage(t, rawdb.PathScheme)
}

func testGenerateBrokenSnapshotWithDanglingStorage(t *testing.T, scheme string) {
	var helper = newHelper(scheme)

	stRoot := helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})

	helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})

	populateDangling(helper.diskdb)

	root, snap := helper.CommitAndGenerate()
	select {
	case <-snap.genPending:
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)

	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}
