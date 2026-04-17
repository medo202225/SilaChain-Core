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
"encoding/json"
"time"

"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/ethdb"
"github.com/SILA/sila-chain/log"
"github.com/SILA/sila-chain/params"
"github.com/SILA/sila-chain/rlp"
)

// ReadDatabaseVersion retrieves the version number of the database.
func ReadDatabaseVersion(db ethdb.KeyValueReader) *uint64 {
var version uint64

enc, _ := db.Get(databaseVersionKey)
if len(enc) == 0 {
return nil
}
if err := rlp.DecodeBytes(enc, &version); err != nil {
return nil
}
return &version
}

// WriteDatabaseVersion stores the version number of the database.
func WriteDatabaseVersion(db ethdb.KeyValueWriter, version uint64) {
enc, err := rlp.EncodeToBytes(version)
if err != nil {
log.Crit("Failed to encode database version", "err", err)
}
if err = db.Put(databaseVersionKey, enc); err != nil {
log.Crit("Failed to store the database version", "err", err)
}
}

// ReadChainConfig retrieves the consensus settings based on the given genesis hash.
func ReadChainConfig(db ethdb.KeyValueReader, hash common.Hash) *params.ChainConfig {
data, _ := db.Get(configKey(hash))
if len(data) == 0 {
return nil
}
var config params.ChainConfig
if err := json.Unmarshal(data, &config); err != nil {
log.Error("Invalid chain config JSON", "hash", hash, "err", err)
return nil
}
return &config
}

// WriteChainConfig writes the chain config settings to the database.
func WriteChainConfig(db ethdb.KeyValueWriter, hash common.Hash, cfg *params.ChainConfig) {
if cfg == nil {
return
}
data, err := json.Marshal(cfg)
if err != nil {
log.Crit("Failed to JSON encode chain config", "err", err)
}
if err := db.Put(configKey(hash), data); err != nil {
log.Crit("Failed to store chain config", "err", err)
}
}

// ReadGenesisStateSpec retrieves the genesis state specification.
func ReadGenesisStateSpec(db ethdb.KeyValueReader, blockhash common.Hash) []byte {
data, _ := db.Get(genesisStateSpecKey(blockhash))
return data
}

// WriteGenesisStateSpec writes the genesis state specification.
func WriteGenesisStateSpec(db ethdb.KeyValueWriter, blockhash common.Hash, data []byte) {
if err := db.Put(genesisStateSpecKey(blockhash), data); err != nil {
log.Crit("Failed to store genesis state", "err", err)
}
}

// crashList is a list of unclean-shutdown-markers.
type crashList struct {
Discarded uint64
Recent    []uint64
}

const crashesToKeep = 10

// PushUncleanShutdownMarker appends a new unclean shutdown marker.
func PushUncleanShutdownMarker(db ethdb.KeyValueStore) ([]uint64, uint64, error) {
var uncleanShutdowns crashList
if data, err := db.Get(uncleanShutdownKey); err == nil {
if err := rlp.DecodeBytes(data, &uncleanShutdowns); err != nil {
return nil, 0, err
}
}
var discarded = uncleanShutdowns.Discarded
var previous = make([]uint64, len(uncleanShutdowns.Recent))
copy(previous, uncleanShutdowns.Recent)

uncleanShutdowns.Recent = append(uncleanShutdowns.Recent, uint64(time.Now().Unix()))
if count := len(uncleanShutdowns.Recent); count > crashesToKeep+1 {
numDel := count - (crashesToKeep + 1)
uncleanShutdowns.Recent = uncleanShutdowns.Recent[numDel:]
uncleanShutdowns.Discarded += uint64(numDel)
}
data, _ := rlp.EncodeToBytes(uncleanShutdowns)
if err := db.Put(uncleanShutdownKey, data); err != nil {
log.Warn("Failed to write unclean-shutdown marker", "err", err)
return nil, 0, err
}
return previous, discarded, nil
}

// PopUncleanShutdownMarker removes the last unclean shutdown marker.
func PopUncleanShutdownMarker(db ethdb.KeyValueStore) {
var uncleanShutdowns crashList
if data, err := db.Get(uncleanShutdownKey); err != nil {
log.Warn("Error reading unclean shutdown markers", "error", err)
} else if err := rlp.DecodeBytes(data, &uncleanShutdowns); err != nil {
log.Error("Error decoding unclean shutdown markers", "error", err)
}
if l := len(uncleanShutdowns.Recent); l > 0 {
uncleanShutdowns.Recent = uncleanShutdowns.Recent[:l-1]
}
data, _ := rlp.EncodeToBytes(uncleanShutdowns)
if err := db.Put(uncleanShutdownKey, data); err != nil {
log.Warn("Failed to clear unclean-shutdown marker", "err", err)
}
}

// UpdateUncleanShutdownMarker updates the last marker'\''s timestamp to now.
func UpdateUncleanShutdownMarker(db ethdb.KeyValueStore) {
var uncleanShutdowns crashList
if data, err := db.Get(uncleanShutdownKey); err != nil {
log.Warn("Error reading unclean shutdown markers", "error", err)
} else if err := rlp.DecodeBytes(data, &uncleanShutdowns); err != nil {
log.Warn("Error decoding unclean shutdown markers", "error", err)
}
count := len(uncleanShutdowns.Recent)
if count == 0 {
log.Warn("No unclean shutdown marker to update")
return
}
uncleanShutdowns.Recent[count-1] = uint64(time.Now().Unix())
data, _ := rlp.EncodeToBytes(uncleanShutdowns)
if err := db.Put(uncleanShutdownKey, data); err != nil {
log.Warn("Failed to write unclean-shutdown marker", "err", err)
}
}
