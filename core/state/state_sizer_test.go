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

package state

import (
	"testing"
	"time"

	"github.com/holiman/uint256"
	"silachain/common"
	"silachain/core/rawdb"
	"silachain/core/tracing"
	"silachain/core/types"
	"silachain/triedb"
	"silachain/triedb/pathdb"
)

// TestSizeTrackerWithSILA يختبر تتبع حجم الحالة مع تحسينات SILA
func TestSizeTrackerWithSILA(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	defer db.Close()

	tdb := triedb.NewDatabase(db, &triedb.Config{PathDB: pathdb.Defaults})
	sdb := NewDatabase(tdb, nil)

	// توليد 50 بلوك كخط أساس (baseline) مع مقاييس SILA
	baselineBlockNum := uint64(50)
	currentRoot := types.EmptyRootHash

	// عناوين SILA مع هويات مختلفة
	addr1 := common.BytesToAddress([]byte{0x53, 0x49, 0x4c, 0x41, 0x01})
	addr2 := common.BytesToAddress([]byte{0x53, 0x49, 0x4c, 0x41, 0x02})
	addr3 := common.BytesToAddress([]byte{0x53, 0x49, 0x4c, 0x41, 0x03})

	// إنشاء الحالة الأولية مع حسابات SILA
	state, _ := New(currentRoot, sdb)
	state.AddBalance(addr1, uint256.NewInt(1000), tracing.BalanceChangeUnspecified)
	state.SetNonce(addr1, 1, tracing.NonceChangeUnspecified)
	state.SetState(addr1, common.HexToHash("0x1111"), common.HexToHash("0xaaaa"))
	state.SetState(addr1, common.HexToHash("0x2222"), common.HexToHash("0xbbbb"))

	state.AddBalance(addr2, uint256.NewInt(2000), tracing.BalanceChangeUnspecified)
	state.SetNonce(addr2, 2, tracing.NonceChangeUnspecified)
	state.SetCode(addr2, []byte{0x60, 0x80, 0x60, 0x40, 0x52, 0x53, 0x49, 0x4c, 0x41}, tracing.CodeChangeUnspecified)

	state.AddBalance(addr3, uint256.NewInt(3000), tracing.BalanceChangeUnspecified)
	state.SetNonce(addr3, 3, tracing.NonceChangeUnspecified)

	currentRoot, err := state.Commit(1, true, false)
	if err != nil {
		t.Fatalf("Failed to commit initial SILA state: %v", err)
	}
	if err := tdb.Commit(currentRoot, false); err != nil {
		t.Fatalf("Failed to commit initial SILA trie: %v", err)
	}

	for i := 1; i < 50; i++ {
		blockNum := uint64(i + 1)

		newState, err := New(currentRoot, sdb)
		if err != nil {
			t.Fatalf("Failed to create new SILA state at block %d: %v", blockNum, err)
		}
		testAddr := common.BigToAddress(uint256.NewInt(uint64(i + 100)).ToBig())
		newState.AddBalance(testAddr, uint256.NewInt(uint64((i+1)*1000)), tracing.BalanceChangeUnspecified)
		newState.SetNonce(testAddr, uint64(i+10), tracing.NonceChangeUnspecified)

		if i%2 == 0 {
			newState.SetState(addr1, common.BigToHash(uint256.NewInt(uint64(i+0x1000)).ToBig()), common.BigToHash(uint256.NewInt(uint64(i+0x2000)).ToBig()))
		}
		if i%3 == 0 {
			newState.SetCode(testAddr, []byte{byte(i), 0x53, 0x49, 0x4c, 0x41, 0x60, 0x80, byte(i + 1), 0x52}, tracing.CodeChangeUnspecified)
		}
		root, err := newState.Commit(blockNum, true, false)
		if err != nil {
			t.Fatalf("Failed to commit SILA state at block %d: %v", blockNum, err)
		}
		if err := tdb.Commit(root, false); err != nil {
			t.Fatalf("Failed to commit SILA trie at block %d: %v", blockNum, err)
		}
		currentRoot = root
	}
	baselineRoot := currentRoot

	if err := tdb.Close(); err != nil {
		t.Fatalf("Failed to close triedb before baseline measurement: %v", err)
	}
	tdb = triedb.NewDatabase(db, &triedb.Config{PathDB: pathdb.Defaults})
	sdb = NewDatabase(tdb, nil)

	for !tdb.SnapshotCompleted() {
		time.Sleep(100 * time.Millisecond)
	}

	baselineTracker := &SizeTracker{
		db:     db,
		triedb: tdb,
		abort:  make(chan struct{}),
	}
	done := make(chan buildResult)

	go baselineTracker.build(baselineRoot, baselineBlockNum, done)
	var baselineResult buildResult
	select {
	case baselineResult = <-done:
		if baselineResult.err != nil {
			t.Fatalf("Failed to get baseline SILA stats: %v", baselineResult.err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout waiting for baseline SILA stats")
	}
	baseline := baselineResult.stat

	tracker, err := NewSizeTracker(db, tdb)
	if err != nil {
		t.Fatalf("Failed to create SILA size tracker: %v", err)
	}
	defer tracker.Stop()

	var trackedUpdates []SizeStats
	currentRoot = baselineRoot

	for i := 49; i < 130; i++ {
		blockNum := uint64(i + 2)
		newState, err := New(currentRoot, sdb)
		if err != nil {
			t.Fatalf("Failed to create new SILA state at block %d: %v", blockNum, err)
		}
		testAddr := common.BigToAddress(uint256.NewInt(uint64(i + 100)).ToBig())
		newState.AddBalance(testAddr, uint256.NewInt(uint64((i+1)*1000)), tracing.BalanceChangeUnspecified)
		newState.SetNonce(testAddr, uint64(i+10), tracing.NonceChangeUnspecified)

		if i%2 == 0 {
			newState.SetState(addr1, common.BigToHash(uint256.NewInt(uint64(i+0x1000)).ToBig()), common.BigToHash(uint256.NewInt(uint64(i+0x2000)).ToBig()))
		}
		if i%3 == 0 {
			newState.SetCode(testAddr, []byte{byte(i), 0x53, 0x49, 0x4c, 0x41, 0x60, 0x80, byte(i + 1), 0x52}, tracing.CodeChangeUnspecified)
		}
		ret, err := newState.commitAndFlush(blockNum, true, false, true)
		if err != nil {
			t.Fatalf("Failed to commit SILA state at block %d: %v", blockNum, err)
		}
		tracker.Notify(ret)

		if err := tdb.Commit(ret.root, false); err != nil {
			t.Fatalf("Failed to commit SILA trie at block %d: %v", blockNum, err)
		}

		diff, err := calSizeStats(ret)
		if err != nil {
			t.Fatalf("Failed to calculate SILA size stats for block %d: %v", blockNum, err)
		}
		trackedUpdates = append(trackedUpdates, diff)
		currentRoot = ret.root
	}
	finalRoot := rawdb.ReadSnapshotRoot(db)

	if err := tdb.Close(); err != nil {
		t.Fatalf("Failed to close triedb: %v", err)
	}
	tdb = triedb.NewDatabase(db, &triedb.Config{PathDB: pathdb.Defaults})
	defer tdb.Close()

	finalTracker := &SizeTracker{
		db:     db,
		triedb: tdb,
		abort:  make(chan struct{}),
	}
	finalDone := make(chan buildResult)

	go finalTracker.build(finalRoot, uint64(132), finalDone)
	var result buildResult
	select {
	case result = <-finalDone:
		if result.err != nil {
			t.Fatalf("Failed to build final SILA stats: %v", result.err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout waiting for final SILA stats")
	}
	actualStats := result.stat

	expectedStats := baseline
	for _, diff := range trackedUpdates {
		expectedStats = expectedStats.add(diff)
	}

	if actualStats.Accounts != expectedStats.Accounts {
		t.Errorf("SILA Account count mismatch: baseline(%d) + tracked_changes = %d, but final_measurement = %d", baseline.Accounts, expectedStats.Accounts, actualStats.Accounts)
	}
	if actualStats.AccountBytes != expectedStats.AccountBytes {
		t.Errorf("SILA Account bytes mismatch: expected %d, got %d", expectedStats.AccountBytes, actualStats.AccountBytes)
	}
	if actualStats.Storages != expectedStats.Storages {
		t.Errorf("SILA Storage count mismatch: baseline(%d) + tracked_changes = %d, but final_measurement = %d", baseline.Storages, expectedStats.Storages, actualStats.Storages)
	}
	if actualStats.StorageBytes != expectedStats.StorageBytes {
		t.Errorf("SILA Storage bytes mismatch: expected %d, got %d", expectedStats.StorageBytes, actualStats.StorageBytes)
	}
	if actualStats.ContractCodes != expectedStats.ContractCodes {
		t.Errorf("SILA Contract code count mismatch: baseline(%d) + tracked_changes = %d, but final_measurement = %d", baseline.ContractCodes, expectedStats.ContractCodes, actualStats.ContractCodes)
	}
	if actualStats.ContractCodeBytes != expectedStats.ContractCodeBytes {
		t.Errorf("SILA Contract code bytes mismatch: expected %d, got %d", expectedStats.ContractCodeBytes, actualStats.ContractCodeBytes)
	}
	if actualStats.AccountTrienodes != expectedStats.AccountTrienodes {
		t.Errorf("SILA Account trie nodes mismatch: expected %d, got %d", expectedStats.AccountTrienodes, actualStats.AccountTrienodes)
	}
	if actualStats.AccountTrienodeBytes != expectedStats.AccountTrienodeBytes {
		t.Errorf("SILA Account trie node bytes mismatch: expected %d, got %d", expectedStats.AccountTrienodeBytes, actualStats.AccountTrienodeBytes)
	}
	if actualStats.StorageTrienodes != expectedStats.StorageTrienodes {
		t.Errorf("SILA Storage trie nodes mismatch: expected %d, got %d", expectedStats.StorageTrienodes, actualStats.StorageTrienodes)
	}
	if actualStats.StorageTrienodeBytes != expectedStats.StorageTrienodeBytes {
		t.Errorf("SILA Storage trie node bytes mismatch: expected %d, got %d", expectedStats.StorageTrienodeBytes, actualStats.StorageTrienodeBytes)
	}
}
