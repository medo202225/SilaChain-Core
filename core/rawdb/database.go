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
"bytes"
"context"
"errors"
"fmt"
"maps"
"os"
"path/filepath"
"runtime"
"slices"
"strings"
"sync"
"sync/atomic"
"time"

"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/crypto"
"github.com/SILA/sila-chain/ethdb"
"github.com/SILA/sila-chain/ethdb/memorydb"
"github.com/SILA/sila-chain/internal/tablewriter"
"github.com/SILA/sila-chain/log"
"golang.org/x/sync/errgroup"
)

var ErrDeleteRangeInterrupted = errors.New("safe delete range operation interrupted")

// freezerdb is a database wrapper that enables ancient chain segment freezing.
type freezerdb struct {
ethdb.KeyValueStore
*chainFreezer

readOnly    bool
ancientRoot string
}

// AncientDatadir returns the path of root ancient directory.
func (frdb *freezerdb) AncientDatadir() (string, error) {
return frdb.ancientRoot, nil
}

// Close implements io.Closer.
func (frdb *freezerdb) Close() error {
var errs []error
if err := frdb.chainFreezer.Close(); err != nil {
errs = append(errs, err)
}
if err := frdb.KeyValueStore.Close(); err != nil {
errs = append(errs, err)
}
if len(errs) != 0 {
return fmt.Errorf("%v", errs)
}
return nil
}

// Freeze is a helper method used for external testing.
func (frdb *freezerdb) Freeze() error {
if frdb.readOnly {
return errReadOnly
}
trigger := make(chan struct{}, 1)
frdb.chainFreezer.trigger <- trigger
<-trigger
return nil
}

// nofreezedb is a database wrapper that disables freezer data retrievals.
type nofreezedb struct {
ethdb.KeyValueStore
}

func (db *nofreezedb) Ancient(kind string, number uint64) ([]byte, error) {
return nil, errNotSupported
}

func (db *nofreezedb) AncientRange(kind string, start, max, maxByteSize uint64) ([][]byte, error) {
return nil, errNotSupported
}

func (db *nofreezedb) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
return nil, errNotSupported
}

func (db *nofreezedb) Ancients() (uint64, error) {
return 0, errNotSupported
}

func (db *nofreezedb) Tail() (uint64, error) {
return 0, errNotSupported
}

func (db *nofreezedb) AncientSize(kind string) (uint64, error) {
return 0, errNotSupported
}

func (db *nofreezedb) ModifyAncients(func(ethdb.AncientWriteOp) error) (int64, error) {
return 0, errNotSupported
}

func (db *nofreezedb) TruncateHead(items uint64) (uint64, error) {
return 0, errNotSupported
}

func (db *nofreezedb) TruncateTail(items uint64) (uint64, error) {
return 0, errNotSupported
}

func (db *nofreezedb) SyncAncient() error {
return errNotSupported
}

func (db *nofreezedb) ReadAncients(fn func(reader ethdb.AncientReaderOp) error) (err error) {
return fn(db)
}

func (db *nofreezedb) AncientDatadir() (string, error) {
return "", errNotSupported
}

// NewDatabase creates a high level database on top of a given key-value store.
func NewDatabase(db ethdb.KeyValueStore) ethdb.Database {
return &nofreezedb{KeyValueStore: db}
}

func resolveChainFreezerDir(ancient string) string {
freezer := filepath.Join(ancient, ChainFreezerName)
if !common.FileExist(freezer) {
if !common.FileExist(ancient) || !common.IsNonEmptyDir(ancient) {
} else {
freezer = ancient
log.Info("Found legacy ancient chain path", "location", ancient)
}
}
return freezer
}

func resolveChainEraDir(chainFreezerDir string, era string) string {
switch {
case era == "":
return filepath.Join(chainFreezerDir, "era")
case !filepath.IsAbs(era):
return filepath.Join(chainFreezerDir, era)
default:
return era
}
}

// NewDatabaseWithFreezer creates a high level database on top of a given key-value store.
// Deprecated: use Open.
func NewDatabaseWithFreezer(db ethdb.KeyValueStore, ancient string, namespace string, readonly bool) (ethdb.Database, error) {
return Open(db, OpenOptions{
Ancient:          ancient,
MetricsNamespace: namespace,
ReadOnly:         readonly,
})
}

// OpenOptions specifies options for opening the database.
type OpenOptions struct {
Ancient          string
Era              string
MetricsNamespace string
ReadOnly         bool
}

// Open creates a high-level database wrapper for the given key-value store.
func Open(db ethdb.KeyValueStore, opts OpenOptions) (ethdb.Database, error) {
chainFreezerDir := opts.Ancient
if chainFreezerDir != "" {
chainFreezerDir = resolveChainFreezerDir(chainFreezerDir)
}
frdb, err := newChainFreezer(chainFreezerDir, opts.Era, opts.MetricsNamespace, opts.ReadOnly)
if err != nil {
printChainMetadata(db)
return nil, err
}

if kvgenesis, _ := db.Get(headerHashKey(0)); len(kvgenesis) > 0 {
if frozen, _ := frdb.Ancients(); frozen > 0 {
frgenesis, err := frdb.Ancient(ChainFreezerHashTable, 0)
if err != nil {
printChainMetadata(db)
return nil, fmt.Errorf("failed to retrieve genesis from ancient %v", err)
} else if !bytes.Equal(kvgenesis, frgenesis) {
printChainMetadata(db)
return nil, fmt.Errorf("genesis mismatch: %#x (leveldb) != %#x (ancients)", kvgenesis, frgenesis)
}
if kvhash, _ := db.Get(headerHashKey(frozen)); len(kvhash) == 0 {
head, ok := ReadHeaderNumber(db, ReadHeadHeaderHash(db))
if !ok {
printChainMetadata(db)
return nil, fmt.Errorf("could not read header number, hash %v", ReadHeadHeaderHash(db))
}
if head > frozen-1 {
var number uint64
for number = frozen; number <= head; number++ {
if present, _ := db.Has(headerHashKey(number)); present {
break
}
}
printChainMetadata(db)
return nil, fmt.Errorf("gap in the chain between ancients [0 - #%d] and leveldb [#%d - #%d] ",
frozen-1, number, head)
}
}
} else {
if ReadHeadHeaderHash(db) != common.BytesToHash(kvgenesis) {
if kvblob, _ := db.Get(headerHashKey(1)); len(kvblob) == 0 {
printChainMetadata(db)
return nil, errors.New("ancient chain segments already extracted, please set --datadir.ancient to the correct path")
}
}
}
}
if !opts.ReadOnly {
frdb.wg.Add(1)
go func() {
frdb.freeze(db)
frdb.wg.Done()
}()
}
return &freezerdb{
readOnly:      opts.ReadOnly,
ancientRoot:   opts.Ancient,
KeyValueStore: db,
chainFreezer:  frdb,
}, nil
}

// NewMemoryDatabase creates an ephemeral in-memory key-value database.
func NewMemoryDatabase() ethdb.Database {
return NewDatabase(memorydb.New())
}

const (
DBPebble  = "pebble"
DBLeveldb = "leveldb"
)

// PreexistingDatabase checks the given data directory whether a database is already instantiated.
func PreexistingDatabase(path string) string {
if _, err := os.Stat(filepath.Join(path, "CURRENT")); err != nil {
return ""
}
if matches, err := filepath.Glob(filepath.Join(path, "OPTIONS*")); len(matches) > 0 || err != nil {
if err != nil {
panic(err)
}
return DBPebble
}
return DBLeveldb
}

type counter uint64

func (c counter) String() string {
return fmt.Sprintf("%d", c)
}

func (c counter) Percentage(current uint64) string {
return fmt.Sprintf("%d", current*100/uint64(c))
}

type stat struct {
size  uint64
count uint64
}

func (s *stat) empty() bool {
return atomic.LoadUint64(&s.count) == 0
}

func (s *stat) add(size common.StorageSize) {
atomic.AddUint64(&s.size, uint64(size))
atomic.AddUint64(&s.count, 1)
}

func (s *stat) sizeString() string {
return common.StorageSize(atomic.LoadUint64(&s.size)).String()
}

func (s *stat) countString() string {
return counter(atomic.LoadUint64(&s.count)).String()
}

// InspectDatabase traverses the entire database and checks the size of all different categories of data.
func InspectDatabase(db ethdb.Database, keyPrefix, keyStart []byte) error {
var (
start = time.Now()
count atomic.Int64
total atomic.Uint64

headers            stat
bodies             stat
receipts           stat
tds                stat
numHashPairings    stat
hashNumPairings    stat
blockAccessList    stat
legacyTries        stat
stateLookups       stat
accountTries       stat
storageTries       stat
codes              stat
txLookups          stat
accountSnaps       stat
storageSnaps       stat
preimages          stat
beaconHeaders      stat
cliqueSnaps        stat
bloomBits          stat
filterMapRows      stat
filterMapLastBlock stat
filterMapBlockLV   stat

stateIndex    stat
trienodeIndex stat

verkleTries        stat
verkleStateLookups stat

metadata    stat
unaccounted stat

unaccountedKeys = make(map[[2]byte][]byte)
unaccountedMu   sync.Mutex
)

inspectRange := func(ctx context.Context, r byte) error {
var s []byte
if len(keyStart) > 0 {
switch {
case r < keyStart[0]:
return nil
case r == keyStart[0]:
s = keyStart[1:]
}
}
it := db.NewIterator(append(keyPrefix, r), s)
defer it.Release()

for it.Next() {
var (
key  = it.Key()
size = common.StorageSize(len(key) + len(it.Value()))
)
total.Add(uint64(size))
count.Add(1)

switch {
case bytes.HasPrefix(key, headerPrefix) && len(key) == (len(headerPrefix)+8+common.HashLength):
headers.add(size)
case bytes.HasPrefix(key, blockBodyPrefix) && len(key) == (len(blockBodyPrefix)+8+common.HashLength):
bodies.add(size)
case bytes.HasPrefix(key, blockReceiptsPrefix) && len(key) == (len(blockReceiptsPrefix)+8+common.HashLength):
receipts.add(size)
case bytes.HasPrefix(key, headerPrefix) && bytes.HasSuffix(key, headerTDSuffix) && len(key) == (len(headerPrefix)+8+common.HashLength+len(headerTDSuffix)):
tds.add(size)
case bytes.HasPrefix(key, headerPrefix) && bytes.HasSuffix(key, headerHashSuffix) && len(key) == (len(headerPrefix)+8+len(headerHashSuffix)):
numHashPairings.add(size)
case bytes.HasPrefix(key, headerNumberPrefix) && len(key) == (len(headerNumberPrefix)+common.HashLength):
hashNumPairings.add(size)
case bytes.HasPrefix(key, accessListPrefix) && len(key) == len(accessListPrefix)+8+common.HashLength:
blockAccessList.add(size)

case IsLegacyTrieNode(key, it.Value()):
legacyTries.add(size)
case bytes.HasPrefix(key, stateIDPrefix) && len(key) == len(stateIDPrefix)+common.HashLength:
stateLookups.add(size)
case IsAccountTrieNode(key):
accountTries.add(size)
case IsStorageTrieNode(key):
storageTries.add(size)
case bytes.HasPrefix(key, CodePrefix) && len(key) == len(CodePrefix)+common.HashLength:
codes.add(size)
case bytes.HasPrefix(key, txLookupPrefix) && len(key) == (len(txLookupPrefix)+common.HashLength):
txLookups.add(size)
case bytes.HasPrefix(key, SnapshotAccountPrefix) && len(key) == (len(SnapshotAccountPrefix)+common.HashLength):
accountSnaps.add(size)
case bytes.HasPrefix(key, SnapshotStoragePrefix) && len(key) == (len(SnapshotStoragePrefix)+2*common.HashLength):
storageSnaps.add(size)
case bytes.HasPrefix(key, PreimagePrefix) && len(key) == (len(PreimagePrefix)+common.HashLength):
preimages.add(size)
case bytes.HasPrefix(key, configPrefix) && len(key) == (len(configPrefix)+common.HashLength):
metadata.add(size)
case bytes.HasPrefix(key, genesisPrefix) && len(key) == (len(genesisPrefix)+common.HashLength):
metadata.add(size)
case bytes.HasPrefix(key, skeletonHeaderPrefix) && len(key) == (len(skeletonHeaderPrefix)+8):
beaconHeaders.add(size)
case bytes.HasPrefix(key, CliqueSnapshotPrefix) && len(key) == 7+common.HashLength:
cliqueSnaps.add(size)

case bytes.HasPrefix(key, filterMapRowPrefix) && len(key) <= len(filterMapRowPrefix)+9:
filterMapRows.add(size)
case bytes.HasPrefix(key, filterMapLastBlockPrefix) && len(key) == len(filterMapLastBlockPrefix)+4:
filterMapLastBlock.add(size)
case bytes.HasPrefix(key, filterMapBlockLVPrefix) && len(key) == len(filterMapBlockLVPrefix)+8:
filterMapBlockLV.add(size)

case bytes.HasPrefix(key, bloomBitsPrefix) && len(key) == (len(bloomBitsPrefix)+10+common.HashLength):
bloomBits.add(size)
case bytes.HasPrefix(key, bloomBitsMetaPrefix) && len(key) < len(bloomBitsMetaPrefix)+8:
bloomBits.add(size)

case bytes.HasPrefix(key, StateHistoryAccountMetadataPrefix) && len(key) == len(StateHistoryAccountMetadataPrefix)+common.HashLength:
stateIndex.add(size)
case bytes.HasPrefix(key, StateHistoryStorageMetadataPrefix) && len(key) == len(StateHistoryStorageMetadataPrefix)+2*common.HashLength:
stateIndex.add(size)
case bytes.HasPrefix(key, StateHistoryAccountBlockPrefix) && len(key) == len(StateHistoryAccountBlockPrefix)+common.HashLength+4:
stateIndex.add(size)
case bytes.HasPrefix(key, StateHistoryStorageBlockPrefix) && len(key) == len(StateHistoryStorageBlockPrefix)+2*common.HashLength+4:
stateIndex.add(size)

case bytes.HasPrefix(key, TrienodeHistoryMetadataPrefix) && len(key) >= len(TrienodeHistoryMetadataPrefix)+common.HashLength:
trienodeIndex.add(size)
case bytes.HasPrefix(key, TrienodeHistoryBlockPrefix) && len(key) >= len(TrienodeHistoryBlockPrefix)+common.HashLength+4:
trienodeIndex.add(size)

case bytes.HasPrefix(key, VerklePrefix):
remain := key[len(VerklePrefix):]
switch {
case IsAccountTrieNode(remain):
verkleTries.add(size)
case bytes.HasPrefix(remain, stateIDPrefix) && len(remain) == len(stateIDPrefix)+common.HashLength:
verkleStateLookups.add(size)
case bytes.Equal(remain, persistentStateIDKey):
metadata.add(size)
case bytes.Equal(remain, trieJournalKey):
metadata.add(size)
case bytes.Equal(remain, snapSyncStatusFlagKey):
metadata.add(size)
default:
unaccounted.add(size)
}

case slices.ContainsFunc(knownMetadataKeys, func(x []byte) bool { return bytes.Equal(x, key) }):
metadata.add(size)

default:
unaccounted.add(size)
if len(key) >= 2 {
prefix := [2]byte(key[:2])
unaccountedMu.Lock()
if _, ok := unaccountedKeys[prefix]; !ok {
unaccountedKeys[prefix] = bytes.Clone(key)
}
unaccountedMu.Unlock()
}
}

select {
case <-ctx.Done():
return ctx.Err()
default:
}
}

return it.Error()
}

var (
eg, ctx = errgroup.WithContext(context.Background())
workers = runtime.NumCPU()
)
eg.SetLimit(workers)

done := make(chan struct{})
go func() {
ticker := time.NewTicker(8 * time.Second)
defer ticker.Stop()

for {
select {
case <-ticker.C:
log.Info("Inspecting database", "count", count.Load(), "size", common.StorageSize(total.Load()), "elapsed", common.PrettyDuration(time.Since(start)))
case <-done:
return
}
}
}()

for i := 0; i < 256; i++ {
eg.Go(func() error { return inspectRange(ctx, byte(i)) })
}

if err := eg.Wait(); err != nil {
close(done)
return err
}
close(done)

stats := [][]string{
{"Key-Value store", "Headers", headers.sizeString(), headers.countString()},
{"Key-Value store", "Bodies", bodies.sizeString(), bodies.countString()},
{"Key-Value store", "Receipt lists", receipts.sizeString(), receipts.countString()},
{"Key-Value store", "Difficulties (deprecated)", tds.sizeString(), tds.countString()},
{"Key-Value store", "Block number->hash", numHashPairings.sizeString(), numHashPairings.countString()},
{"Key-Value store", "Block hash->number", hashNumPairings.sizeString(), hashNumPairings.countString()},
{"Key-Value store", "Block accessList", blockAccessList.sizeString(), blockAccessList.countString()},
{"Key-Value store", "Transaction index", txLookups.sizeString(), txLookups.countString()},
{"Key-Value store", "Log index filter-map rows", filterMapRows.sizeString(), filterMapRows.countString()},
{"Key-Value store", "Log index last-block-of-map", filterMapLastBlock.sizeString(), filterMapLastBlock.countString()},
{"Key-Value store", "Log index block-lv", filterMapBlockLV.sizeString(), filterMapBlockLV.countString()},
{"Key-Value store", "Log bloombits (deprecated)", bloomBits.sizeString(), bloomBits.countString()},
{"Key-Value store", "Contract codes", codes.sizeString(), codes.countString()},
{"Key-Value store", "Hash trie nodes", legacyTries.sizeString(), legacyTries.countString()},
{"Key-Value store", "Path trie state lookups", stateLookups.sizeString(), stateLookups.countString()},
{"Key-Value store", "Path trie account nodes", accountTries.sizeString(), accountTries.countString()},
{"Key-Value store", "Path trie storage nodes", storageTries.sizeString(), storageTries.countString()},
{"Key-Value store", "Verkle trie nodes", verkleTries.sizeString(), verkleTries.countString()},
{"Key-Value store", "Verkle trie state lookups", verkleStateLookups.sizeString(), verkleStateLookups.countString()},
{"Key-Value store", "Trie preimages", preimages.sizeString(), preimages.countString()},
{"Key-Value store", "Account snapshot", accountSnaps.sizeString(), accountSnaps.countString()},
{"Key-Value store", "Storage snapshot", storageSnaps.sizeString(), storageSnaps.countString()},
{"Key-Value store", "Historical state index", stateIndex.sizeString(), stateIndex.countString()},
{"Key-Value store", "Historical trie index", trienodeIndex.sizeString(), trienodeIndex.countString()},
{"Key-Value store", "Beacon sync headers", beaconHeaders.sizeString(), beaconHeaders.countString()},
{"Key-Value store", "Clique snapshots", cliqueSnaps.sizeString(), cliqueSnaps.countString()},
{"Key-Value store", "Singleton metadata", metadata.sizeString(), metadata.countString()},
}

ancients, err := inspectFreezers(db)
if err != nil {
return err
}
for _, ancient := range ancients {
for _, table := range ancient.sizes {
stats = append(stats, []string{
fmt.Sprintf("Ancient store (%s)", strings.Title(ancient.name)),
strings.Title(table.name),
table.size.String(),
fmt.Sprintf("%d", ancient.count),
})
}
total.Add(uint64(ancient.size()))
}

table := tablewriter.NewWriter(os.Stdout)
table.SetHeader([]string{"Database", "Category", "Size", "Items"})
table.SetFooter([]string{"", "Total", common.StorageSize(total.Load()).String(), fmt.Sprintf("%d", count.Load())})
table.AppendBulk(stats)
table.Render()

if !unaccounted.empty() {
log.Error("Database contains unaccounted data", "size", unaccounted.sizeString(), "count", unaccounted.countString())
for _, e := range slices.SortedFunc(maps.Values(unaccountedKeys), bytes.Compare) {
log.Error(fmt.Sprintf("   example key: %x", e))
}
}
return nil
}

var knownMetadataKeys = [][]byte{
databaseVersionKey, headHeaderKey, headBlockKey, headFastBlockKey, headFinalizedBlockKey,
lastPivotKey, fastTrieProgressKey, snapshotDisabledKey, SnapshotRootKey, snapshotJournalKey,
snapshotGeneratorKey, snapshotRecoveryKey, txIndexTailKey, fastTxLookupLimitKey,
uncleanShutdownKey, badBlockKey, transitionStatusKey, skeletonSyncStatusKey,
persistentStateIDKey, trieJournalKey, snapshotSyncStatusKey, snapSyncStatusFlagKey,
filterMapsRangeKey, headStateHistoryIndexKey, headTrienodeHistoryIndexKey, VerkleTransitionStatePrefix,
}

func printChainMetadata(db ethdb.KeyValueStore) {
fmt.Fprintf(os.Stderr, "Chain metadata\n")
for _, v := range ReadChainMetadata(db) {
fmt.Fprintf(os.Stderr, "  %s\n", strings.Join(v, ": "))
}
fmt.Fprintf(os.Stderr, "\n\n")
}

// ReadChainMetadata returns a set of key/value pairs that contains information about the database chain status.
func ReadChainMetadata(db ethdb.KeyValueStore) [][]string {
pp := func(val *uint64) string {
if val == nil {
return "<nil>"
}
return fmt.Sprintf("%d (%#x)", *val, *val)
}

data := [][]string{
{"databaseVersion", pp(ReadDatabaseVersion(db))},
{"headBlockHash", fmt.Sprintf("%v", ReadHeadBlockHash(db))},
{"headFastBlockHash", fmt.Sprintf("%v", ReadHeadFastBlockHash(db))},
{"headHeaderHash", fmt.Sprintf("%v", ReadHeadHeaderHash(db))},
{"lastPivotNumber", pp(ReadLastPivotNumber(db))},
{"len(snapshotSyncStatus)", fmt.Sprintf("%d bytes", len(ReadSnapshotSyncStatus(db)))},
{"snapshotDisabled", fmt.Sprintf("%v", ReadSnapshotDisabled(db))},
{"snapshotJournal", fmt.Sprintf("%d bytes", len(ReadSnapshotJournal(db)))},
{"snapshotRecoveryNumber", pp(ReadSnapshotRecoveryNumber(db))},
{"snapshotRoot", fmt.Sprintf("%v", ReadSnapshotRoot(db))},
{"txIndexTail", pp(ReadTxIndexTail(db))},
}
if b := ReadSkeletonSyncStatus(db); b != nil {
data = append(data, []string{"SkeletonSyncStatus", string(b)})
}
if fmr, ok, _ := ReadFilterMapsRange(db); ok {
data = append(data, []string{"filterMapsRange", fmt.Sprintf("%+v", fmr)})
}
return data
}

// SafeDeleteRange deletes all of the keys (and values) in the range [start,end).
func SafeDeleteRange(db ethdb.KeyValueStore, start, end []byte, hashScheme bool, stopCallback func(bool) bool) error {
if !hashScheme {
for {
switch err := db.DeleteRange(start, end); {
case err == nil:
return nil
case errors.Is(err, ethdb.ErrTooManyKeys):
if stopCallback(true) {
return ErrDeleteRangeInterrupted
}
default:
return err
}
}
}

var (
count, deleted, skipped int
startTime               = time.Now()
)

batch := db.NewBatch()
it := db.NewIterator(nil, start)
defer func() {
it.Release()
log.Debug("SafeDeleteRange finished", "deleted", deleted, "skipped", skipped, "elapsed", common.PrettyDuration(time.Since(startTime)))
}()

for it.Next() && bytes.Compare(end, it.Key()) > 0 {
if len(it.Key()) != 32 || crypto.Keccak256Hash(it.Value()) != common.BytesToHash(it.Key()) {
if err := batch.Delete(it.Key()); err != nil {
return err
}
deleted++
} else {
skipped++
}
count++
if count > 10000 {
if err := batch.Write(); err != nil {
return err
}
if stopCallback(deleted != 0) {
return ErrDeleteRangeInterrupted
}
start = append(bytes.Clone(it.Key()), 0)
it.Release()
batch = db.NewBatch()
it = db.NewIterator(nil, start)
count = 0
}
}
return batch.Write()
}
