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
"bytes"
crand "crypto/rand"
"math/rand"
"testing"

"github.com/VictoriaMetrics/fastcache"
"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/crypto"
"github.com/SILA/sila-chain/ethdb/memorydb"
)

func copyAccounts(accounts map[common.Hash][]byte) map[common.Hash][]byte {
copy := make(map[common.Hash][]byte)
for hash, blob := range accounts {
copy[hash] = blob
}
return copy
}

func copyStorage(storage map[common.Hash]map[common.Hash][]byte) map[common.Hash]map[common.Hash][]byte {
copy := make(map[common.Hash]map[common.Hash][]byte)
for accHash, slots := range storage {
copy[accHash] = make(map[common.Hash][]byte)
for slotHash, blob := range slots {
copy[accHash][slotHash] = blob
}
}
return copy
}

// TestMergeBasics tests some simple merges.
func TestMergeBasics(t *testing.T) {
var (
accounts = make(map[common.Hash][]byte)
storage  = make(map[common.Hash]map[common.Hash][]byte)
)
for i := 0; i < 100; i++ {
h := randomHash()
data := randomAccount()

accounts[h] = data
if rand.Intn(4) == 0 {
accounts[h] = nil
}
if rand.Intn(2) == 0 {
accStorage := make(map[common.Hash][]byte)
value := make([]byte, 32)
crand.Read(value)
accStorage[randomHash()] = value
storage[h] = accStorage
}
}
parent := newDiffLayer(emptyLayer(), common.Hash{}, copyAccounts(accounts), copyStorage(storage))
child := newDiffLayer(parent, common.Hash{}, copyAccounts(accounts), copyStorage(storage))
child = newDiffLayer(child, common.Hash{}, copyAccounts(accounts), copyStorage(storage))
child = newDiffLayer(child, common.Hash{}, copyAccounts(accounts), copyStorage(storage))
child = newDiffLayer(child, common.Hash{}, copyAccounts(accounts), copyStorage(storage))

merged := (child.flatten()).(*diffLayer)

{
if have, want := len(merged.accountList), 0; have != want {
t.Errorf("accountList wrong: have %v, want %v", have, want)
}
if have, want := len(merged.AccountList()), len(accounts); have != want {
t.Errorf("AccountList() wrong: have %v, want %v", have, want)
}
if have, want := len(merged.accountList), len(accounts); have != want {
t.Errorf("accountList [2] wrong: have %v, want %v", have, want)
}
}
{
i := 0
for aHash, sMap := range storage {
if have, want := len(merged.storageList), i; have != want {
t.Errorf("[1] storageList wrong: have %v, want %v", have, want)
}
list := merged.StorageList(aHash)
if have, want := len(list), len(sMap); have != want {
t.Errorf("[2] StorageList() wrong: have %v, want %v", have, want)
}
if have, want := len(merged.storageList[aHash]), len(sMap); have != want {
t.Errorf("storageList wrong: have %v, want %v", have, want)
}
i++
}
}
}

// TestMergeDelete tests some deletion.
func TestMergeDelete(t *testing.T) {
storage := make(map[common.Hash]map[common.Hash][]byte)

h1 := common.HexToHash("0x01")
h2 := common.HexToHash("0x02")

flip := func() map[common.Hash][]byte {
return map[common.Hash][]byte{
h1: randomAccount(),
h2: nil,
}
}
flop := func() map[common.Hash][]byte {
return map[common.Hash][]byte{
h1: nil,
h2: randomAccount(),
}
}
parent := newDiffLayer(emptyLayer(), common.Hash{}, flip(), storage)
child := parent.Update(common.Hash{}, flop(), storage)
child = child.Update(common.Hash{}, flip(), storage)
child = child.Update(common.Hash{}, flop(), storage)
child = child.Update(common.Hash{}, flip(), storage)
child = child.Update(common.Hash{}, flop(), storage)
child = child.Update(common.Hash{}, flip(), storage)

if data, _ := child.Account(h1); data == nil {
t.Errorf("last diff layer: expected %x account to be non-nil", h1)
}
if data, _ := child.Account(h2); data != nil {
t.Errorf("last diff layer: expected %x account to be nil", h2)
}

merged := (child.flatten()).(*diffLayer)

if data, _ := merged.Account(h1); data == nil {
t.Errorf("merged layer: expected %x account to be non-nil", h1)
}
if data, _ := merged.Account(h2); data != nil {
t.Errorf("merged layer: expected %x account to be nil", h2)
}
}

// TestInsertAndMerge tests creating a new account and slot, then merging.
func TestInsertAndMerge(t *testing.T) {
var (
acc    = common.HexToHash("0x01")
slot   = common.HexToHash("0x02")
parent *diffLayer
child  *diffLayer
)
{
var (
accounts = make(map[common.Hash][]byte)
storage  = make(map[common.Hash]map[common.Hash][]byte)
)
parent = newDiffLayer(emptyLayer(), common.Hash{}, accounts, storage)
}
{
var (
accounts = make(map[common.Hash][]byte)
storage  = make(map[common.Hash]map[common.Hash][]byte)
)
accounts[acc] = randomAccount()
storage[acc] = make(map[common.Hash][]byte)
storage[acc][slot] = []byte{0x01}
child = newDiffLayer(parent, common.Hash{}, accounts, storage)
}
merged := (child.flatten()).(*diffLayer)
{
have, _ := merged.Storage(acc, slot)
if want := []byte{0x01}; !bytes.Equal(have, want) {
t.Errorf("merged slot value wrong: have %x, want %x", have, want)
}
}
}

// TestStorageListMemoryAccounting ensures that StorageList increases dl.memory proportionally.
func TestStorageListMemoryAccounting(t *testing.T) {
parent := newDiffLayer(emptyLayer(), common.Hash{}, nil, nil)
account := common.HexToHash("0x01")

slots := make(map[common.Hash][]byte)
for i := 0; i < 3; i++ {
slots[randomHash()] = []byte{0x01}
}
storage := map[common.Hash]map[common.Hash][]byte{
account: slots,
}
dl := newDiffLayer(parent, common.Hash{}, nil, storage)

before := dl.memory
list := dl.StorageList(account)
if have, want := len(list), len(slots); have != want {
t.Fatalf("StorageList length mismatch: have %d, want %d", have, want)
}
expectedDelta := uint64(len(list)*common.HashLength + common.HashLength)
if have, want := dl.memory-before, expectedDelta; have != want {
t.Fatalf("StorageList memory delta mismatch: have %d, want %d", have, want)
}

before = dl.memory
_ = dl.StorageList(account)
if dl.memory != before {
t.Fatalf("StorageList changed memory on cached call: have %d, want %d", dl.memory, before)
}
}

func emptyLayer() *diskLayer {
return &diskLayer{
diskdb: memorydb.New(),
cache:  fastcache.New(500 * 1024),
}
}

// BenchmarkSearch checks how long it takes to find a non-existing key.
func BenchmarkSearch(b *testing.B) {
fill := func(parent snapshot) *diffLayer {
var (
accounts = make(map[common.Hash][]byte)
storage  = make(map[common.Hash]map[common.Hash][]byte)
)
for i := 0; i < 10000; i++ {
accounts[randomHash()] = randomAccount()
}
return newDiffLayer(parent, common.Hash{}, accounts, storage)
}
var layer snapshot
layer = emptyLayer()
for i := 0; i < 128; i++ {
layer = fill(layer)
}
key := crypto.Keccak256Hash([]byte{0x13, 0x38})
for b.Loop() {
layer.AccountRLP(key)
}
}

// BenchmarkSearchSlot checks how long it takes to find a non-existing storage slot.
func BenchmarkSearchSlot(b *testing.B) {
accountKey := crypto.Keccak256Hash([]byte{0x13, 0x37})
storageKey := crypto.Keccak256Hash([]byte{0x13, 0x37})
accountRLP := randomAccount()
fill := func(parent snapshot) *diffLayer {
var (
accounts = make(map[common.Hash][]byte)
storage  = make(map[common.Hash]map[common.Hash][]byte)
)
accounts[accountKey] = accountRLP

accStorage := make(map[common.Hash][]byte)
for i := 0; i < 5; i++ {
value := make([]byte, 32)
crand.Read(value)
accStorage[randomHash()] = value
storage[accountKey] = accStorage
}
return newDiffLayer(parent, common.Hash{}, accounts, storage)
}
var layer snapshot
layer = emptyLayer()
for i := 0; i < 128; i++ {
layer = fill(layer)
}
for b.Loop() {
layer.Storage(accountKey, storageKey)
}
}

// BenchmarkFlatten benchmarks the flatten operation.
func BenchmarkFlatten(b *testing.B) {
fill := func(parent snapshot) *diffLayer {
var (
accounts = make(map[common.Hash][]byte)
storage  = make(map[common.Hash]map[common.Hash][]byte)
)
for i := 0; i < 100; i++ {
accountKey := randomHash()
accounts[accountKey] = randomAccount()

accStorage := make(map[common.Hash][]byte)
for i := 0; i < 20; i++ {
value := make([]byte, 32)
crand.Read(value)
accStorage[randomHash()] = value
}
storage[accountKey] = accStorage
}
return newDiffLayer(parent, common.Hash{}, accounts, storage)
}
for b.Loop() {
var layer snapshot
layer = emptyLayer()
for i := 1; i < 128; i++ {
layer = fill(layer)
}
b.StartTimer()

for i := 1; i < 128; i++ {
dl, ok := layer.(*diffLayer)
if !ok {
break
}
layer = dl.flatten()
}
b.StopTimer()
}
}

// BenchmarkJournal benchmarks the journal operation.
func BenchmarkJournal(b *testing.B) {
fill := func(parent snapshot) *diffLayer {
var (
accounts = make(map[common.Hash][]byte)
storage  = make(map[common.Hash]map[common.Hash][]byte)
)
for i := 0; i < 200; i++ {
accountKey := randomHash()
accounts[accountKey] = randomAccount()

accStorage := make(map[common.Hash][]byte)
for i := 0; i < 200; i++ {
value := make([]byte, 32)
crand.Read(value)
accStorage[randomHash()] = value
}
storage[accountKey] = accStorage
}
return newDiffLayer(parent, common.Hash{}, accounts, storage)
}
layer := snapshot(emptyLayer())
for i := 1; i < 128; i++ {
layer = fill(layer)
}
for b.Loop() {
layer.Journal(new(bytes.Buffer))
}
}
