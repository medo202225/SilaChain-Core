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

package filtermaps

import (
crand "crypto/rand"
"math/rand"
"testing"

"github.com/SILA/sila-chain/common"
)

func TestSingleMatch(t *testing.T) {
params := DefaultParams
params.deriveFields()

for count := 0; count < 100000; count++ {
mapIndex := rand.Uint32()
lvIndex := uint64(mapIndex)<<params.logValuesPerMap + uint64(rand.Intn(int(params.valuesPerMap)))
var lvHash common.Hash
crand.Read(lvHash[:])
row := FilterRow{params.columnIndex(lvIndex, &lvHash)}
matches := params.potentialMatches([]FilterRow{row}, mapIndex, lvHash)
if len(matches) != 1 {
t.Fatalf("Invalid length of matches (got %d, expected 1)", len(matches))
}
if matches[0] != lvIndex {
t.Fatalf("Incorrect match returned (got %d, expected %d)", matches[0], lvIndex)
}
}
}

const (
testPmCount = 50
testPmLen   = 1000
)

func TestPotentialMatches(t *testing.T) {
params := DefaultParams
params.deriveFields()

var falsePositives int
for count := 0; count < testPmCount; count++ {
mapIndex := rand.Uint32()
lvStart := uint64(mapIndex) << params.logValuesPerMap
var row FilterRow
lvIndices := make([]uint64, testPmLen)
lvHashes := make([]common.Hash, testPmLen+1)
for i := range lvIndices {
lvIndices[i] = lvStart + uint64(rand.Intn(int(params.valuesPerMap)))
crand.Read(lvHashes[i][:])
row = append(row, params.columnIndex(lvIndices[i], &lvHashes[i]))
}
crand.Read(lvHashes[testPmLen][:])
for lvIndex := lvStart; lvIndex < lvStart+testPmLen; lvIndex++ {
row = append(row, params.columnIndex(lvIndex, &lvHashes[testPmLen]))
}
for i := 0; i < testPmLen; i++ {
row = append(row, row[rand.Intn(len(row))])
}
for i := len(row) - 1; i > 0; i-- {
j := rand.Intn(i)
row[i], row[j] = row[j], row[i]
}
var rows []FilterRow
for layerIndex := uint32(0); row != nil; layerIndex++ {
maxLen := int(params.maxRowLength(layerIndex))
if len(row) > maxLen {
rows = append(rows, row[:maxLen])
row = row[maxLen:]
} else {
rows = append(rows, row)
row = nil
}
}
for i, lvHash := range lvHashes {
matches := params.potentialMatches(rows, mapIndex, lvHash)
if i < testPmLen {
if len(matches) < 1 {
t.Fatalf("Invalid length of matches (got %d, expected >=1)", len(matches))
}
var found bool
for _, lvi := range matches {
if lvi == lvIndices[i] {
found = true
} else {
falsePositives++
}
}
if !found {
t.Fatalf("Expected match not found (got %v, expected %d)", matches, lvIndices[i])
}
} else {
if len(matches) < testPmLen {
t.Fatalf("Invalid length of matches (got %d, expected >=%d)", len(matches), testPmLen)
}
for j := 0; j < testPmLen; j++ {
if matches[j] != lvStart+uint64(j) {
t.Fatalf("Incorrect match at index %d (got %d, expected %d)", j, matches[j], lvStart+uint64(j))
}
}
falsePositives += len(matches) - testPmLen
}
}
}
expFalse := int(uint64(testPmCount*testPmLen*testPmLen*2) * params.valuesPerMap >> params.logMapWidth)
if falsePositives < expFalse/2 || falsePositives > expFalse*3/2 {
t.Fatalf("False positive rate out of expected range (got %d, expected %d +-50%%)", falsePositives, expFalse)
}
}
