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

package pruner

import (
"bytes"
"encoding/binary"
"errors"
"fmt"
"math"
"os"
"path/filepath"
"strings"
"time"

"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/core/rawdb"
"github.com/SILA/sila-chain/core/state/snapshot"
"github.com/SILA/sila-chain/core/types"
"github.com/SILA/sila-chain/ethdb"
"github.com/SILA/sila-chain/log"
"github.com/SILA/sila-chain/rlp"
"github.com/SILA/sila-chain/trie"
"github.com/SILA/sila-chain/triedb"
)

const (
stateBloomFilePrefix      = "statebloom"
stateBloomFileSuffix      = "bf.gz"
stateBloomFileTempSuffix  = ".tmp"
rangeCompactionThreshold  = 100000
)

// Config includes all the configurations for pruning.
type Config struct {
Datadir   string
BloomSize uint64
}

// Pruner is an offline tool to prune the stale state.
type Pruner struct {
config      Config
chainHeader *types.Header
db          ethdb.Database
stateBloom  *stateBloom
snaptree    *snapshot.Tree
}

// NewPruner creates the pruner instance.
func NewPruner(db ethdb.Database, config Config) (*Pruner, error) {
headBlock := rawdb.ReadHeadBlock(db)
if headBlock == nil {
return nil, errors.New("failed to load head block")
}
triedb := triedb.NewDatabase(db, triedb.HashDefaults)

snapconfig := snapshot.Config{
CacheSize:  256,
Recovery:   false,
NoBuild:    true,
AsyncBuild: false,
}
snaptree, err := snapshot.New(snapconfig, db, triedb, headBlock.Root())
if err != nil {
return nil, err
}
if config.BloomSize < 256 {
log.Warn("Sanitizing bloomfilter size", "provided(MB)", config.BloomSize, "updated(MB)", 256)
config.BloomSize = 256
}
stateBloom, err := newStateBloomWithSize(config.BloomSize)
if err != nil {
return nil, err
}
return &Pruner{
config:      config,
chainHeader: headBlock.Header(),
db:          db,
stateBloom:  stateBloom,
snaptree:    snaptree,
}, nil
}

func prune(snaptree *snapshot.Tree, root common.Hash, maindb ethdb.Database, stateBloom *stateBloom, bloomPath string, middleStateRoots map[common.Hash]struct{}, start time.Time) error {
var (
skipped, count int
size           common.StorageSize
pstart         = time.Now()
logged         = time.Now()
batch          = maindb.NewBatch()
iter           = maindb.NewIterator(nil, nil)
)
for iter.Next() {
key := iter.Key()

isCode, codeKey := rawdb.IsCodeKey(key)
if len(key) == common.HashLength || isCode {
checkKey := key
if isCode {
checkKey = codeKey
}
if _, exist := middleStateRoots[common.BytesToHash(checkKey)]; exist {
log.Debug("Forcibly delete the middle state roots", "hash", common.BytesToHash(checkKey))
} else {
if stateBloom.Contain(checkKey) {
skipped += 1
continue
}
}
count += 1
size += common.StorageSize(len(key) + len(iter.Value()))
batch.Delete(key)

var eta time.Duration
if done := binary.BigEndian.Uint64(key[:8]); done > 0 {
left := math.MaxUint64 - binary.BigEndian.Uint64(key[:8])
eta = common.CalculateETA(done, left, time.Since(pstart))
}
if time.Since(logged) > 8*time.Second {
log.Info("Pruning state data", "nodes", count, "skipped", skipped, "size", size,
"elapsed", common.PrettyDuration(time.Since(pstart)), "eta", common.PrettyDuration(eta))
logged = time.Now()
}
if batch.ValueSize() >= ethdb.IdealBatchSize {
batch.Write()
batch.Reset()

iter.Release()
iter = maindb.NewIterator(nil, key)
}
}
}
if batch.ValueSize() > 0 {
batch.Write()
batch.Reset()
}
iter.Release()
log.Info("Pruned state data", "nodes", count, "size", size, "elapsed", common.PrettyDuration(time.Since(pstart)))

if err := snaptree.Cap(root, 0); err != nil {
return err
}
if _, err := snaptree.Journal(root); err != nil {
return err
}
os.RemoveAll(bloomPath)

if count >= rangeCompactionThreshold {
cstart := time.Now()
for b := 0x00; b <= 0xf0; b += 0x10 {
var (
start = []byte{byte(b)}
end   = []byte{byte(b + 0x10)}
)
if b == 0xf0 {
end = nil
}
log.Info("Compacting database", "range", fmt.Sprintf("%#x-%#x", start, end), "elapsed", common.PrettyDuration(time.Since(cstart)))
if err := maindb.Compact(start, end); err != nil {
log.Error("Database compaction failed", "error", err)
return err
}
}
log.Info("Database compaction finished", "elapsed", common.PrettyDuration(time.Since(cstart)))
}
log.Info("State pruning successful", "pruned", size, "elapsed", common.PrettyDuration(time.Since(start)))
return nil
}

// Prune deletes all historical state nodes except the nodes belong to the specified state version.
func (p *Pruner) Prune(root common.Hash) error {
_, stateBloomRoot, err := findBloomFilter(p.config.Datadir)
if err != nil {
return err
}
if stateBloomRoot != (common.Hash{}) {
return RecoverPruning(p.config.Datadir, p.db)
}
var layers []snapshot.Snapshot
if root == (common.Hash{}) {
layers = p.snaptree.Snapshots(p.chainHeader.Root, 128, true)
if len(layers) != 128 {
return fmt.Errorf("snapshot not old enough yet: need %d more blocks", 128-len(layers))
}
root = layers[len(layers)-1].Root()
}
if !rawdb.HasLegacyTrieNode(p.db, root) {
var found bool
for i := len(layers) - 2; i >= 2; i-- {
if rawdb.HasLegacyTrieNode(p.db, layers[i].Root()) {
root = layers[i].Root()
found = true
log.Info("Selecting middle-layer as the pruning target", "root", root, "depth", i)
break
}
}
if !found {
if len(layers) > 0 {
return errors.New("no snapshot paired state")
}
return fmt.Errorf("associated state[%x] is not present", root)
}
} else {
if len(layers) > 0 {
log.Info("Selecting bottom-most difflayer as the pruning target", "root", root, "height", p.chainHeader.Number.Uint64()-127)
} else {
log.Info("Selecting user-specified state as the pruning target", "root", root)
}
}
middleRoots := make(map[common.Hash]struct{})
for _, layer := range layers {
if layer.Root() == root {
break
}
middleRoots[layer.Root()] = struct{}{}
}
start := time.Now()
if err := snapshot.GenerateTrie(p.snaptree, root, p.db, p.stateBloom); err != nil {
return err
}
if err := extractGenesis(p.db, p.stateBloom); err != nil {
return err
}
filterName := bloomFilterName(p.config.Datadir, root)

log.Info("Writing state bloom to disk", "name", filterName)
if err := p.stateBloom.Commit(filterName, filterName+stateBloomFileTempSuffix); err != nil {
return err
}
log.Info("State bloom filter committed", "name", filterName)
return prune(p.snaptree, root, p.db, p.stateBloom, filterName, middleRoots, start)
}

// RecoverPruning will resume the pruning procedure during the system restart.
func RecoverPruning(datadir string, db ethdb.Database) error {
stateBloomPath, stateBloomRoot, err := findBloomFilter(datadir)
if err != nil {
return err
}
if stateBloomPath == "" {
return nil
}
headBlock := rawdb.ReadHeadBlock(db)
if headBlock == nil {
return errors.New("failed to load head block")
}
snapconfig := snapshot.Config{
CacheSize:  256,
Recovery:   true,
NoBuild:    true,
AsyncBuild: false,
}
triedb := triedb.NewDatabase(db, triedb.HashDefaults)
snaptree, err := snapshot.New(snapconfig, db, triedb, headBlock.Root())
if err != nil {
return err
}
stateBloom, err := NewStateBloomFromDisk(stateBloomPath)
if err != nil {
return err
}
log.Info("Loaded state bloom filter", "path", stateBloomPath)

var (
found       bool
layers      = snaptree.Snapshots(headBlock.Root(), 128, true)
middleRoots = make(map[common.Hash]struct{})
)
for _, layer := range layers {
if layer.Root() == stateBloomRoot {
found = true
break
}
middleRoots[layer.Root()] = struct{}{}
}
if !found {
log.Error("Pruning target state is not existent")
return errors.New("non-existent target state")
}
return prune(snaptree, stateBloomRoot, db, stateBloom, stateBloomPath, middleRoots, time.Now())
}

// extractGenesis loads the genesis state and commits all the state entries into the bloomfilter.
func extractGenesis(db ethdb.Database, stateBloom *stateBloom) error {
genesisHash := rawdb.ReadCanonicalHash(db, 0)
if genesisHash == (common.Hash{}) {
return errors.New("missing genesis hash")
}
genesis := rawdb.ReadBlock(db, genesisHash, 0)
if genesis == nil {
return errors.New("missing genesis block")
}
t, err := trie.NewStateTrie(trie.StateTrieID(genesis.Root()), triedb.NewDatabase(db, triedb.HashDefaults))
if err != nil {
return err
}
accIter, err := t.NodeIterator(nil)
if err != nil {
return err
}
for accIter.Next(true) {
hash := accIter.Hash()

if hash != (common.Hash{}) {
stateBloom.Put(hash.Bytes(), nil)
}
if accIter.Leaf() {
var acc types.StateAccount
if err := rlp.DecodeBytes(accIter.LeafBlob(), &acc); err != nil {
return err
}
if acc.Root != types.EmptyRootHash {
id := trie.StorageTrieID(genesis.Root(), common.BytesToHash(accIter.LeafKey()), acc.Root)
storageTrie, err := trie.NewStateTrie(id, triedb.NewDatabase(db, triedb.HashDefaults))
if err != nil {
return err
}
storageIter, err := storageTrie.NodeIterator(nil)
if err != nil {
return err
}
for storageIter.Next(true) {
hash := storageIter.Hash()
if hash != (common.Hash{}) {
stateBloom.Put(hash.Bytes(), nil)
}
}
if storageIter.Error() != nil {
return storageIter.Error()
}
}
if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash.Bytes()) {
stateBloom.Put(acc.CodeHash, nil)
}
}
}
return accIter.Error()
}

func bloomFilterName(datadir string, hash common.Hash) string {
return filepath.Join(datadir, fmt.Sprintf("%s.%s.%s", stateBloomFilePrefix, hash.Hex(), stateBloomFileSuffix))
}

func isBloomFilter(filename string) (bool, common.Hash) {
filename = filepath.Base(filename)
if strings.HasPrefix(filename, stateBloomFilePrefix) && strings.HasSuffix(filename, stateBloomFileSuffix) {
return true, common.HexToHash(filename[len(stateBloomFilePrefix)+1 : len(filename)-len(stateBloomFileSuffix)-1])
}
return false, common.Hash{}
}

func findBloomFilter(datadir string) (string, common.Hash, error) {
var (
stateBloomPath string
stateBloomRoot common.Hash
)
if err := filepath.Walk(datadir, func(path string, info os.FileInfo, err error) error {
if info != nil && !info.IsDir() {
ok, root := isBloomFilter(path)
if ok {
stateBloomPath = path
stateBloomRoot = root
}
}
return nil
}); err != nil {
return "", common.Hash{}, err
}
return stateBloomPath, stateBloomRoot, nil
}
