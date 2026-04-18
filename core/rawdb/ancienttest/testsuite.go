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

package ancienttest

import (
"bytes"
"reflect"
"testing"

"silachain/ethdb"
"silachain/internal/testrand"
)

// TestAncientSuite runs a suite of tests against an ancient database
// implementation.
func TestAncientSuite(t *testing.T, newFn func(kinds []string) ethdb.AncientStore) {
t.Run("BasicRead", func(t *testing.T) { basicRead(t, newFn) })
t.Run("BatchRead", func(t *testing.T) { batchRead(t, newFn) })
t.Run("BasicWrite", func(t *testing.T) { basicWrite(t, newFn) })
t.Run("nonMutable", func(t *testing.T) { nonMutable(t, newFn) })
}

func basicRead(t *testing.T, newFn func(kinds []string) ethdb.AncientStore) {
var (
db   = newFn([]string{"a"})
data = makeDataset(100, 32)
)
defer db.Close()

if _, err := db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
for i := 0; i < len(data); i++ {
if err := op.AppendRaw("a", uint64(i), data[i]); err != nil {
return err
}
}
return nil
}); err != nil {
t.Fatalf("Failed to write ancient data %v", err)
}
db.TruncateTail(10)
db.TruncateHead(90)

tail, err := db.Tail()
if err != nil || tail != 10 {
t.Fatal("Failed to retrieve tail")
}
ancient, err := db.Ancients()
if err != nil || ancient != 90 {
t.Fatal("Failed to retrieve ancient")
}

var cases = []struct {
start int
limit int
}{
{0, 10},
{90, 100},
}
for _, c := range cases {
for i := c.start; i < c.limit; i++ {
_, err = db.Ancient("a", uint64(i))
if err == nil {
t.Fatal("Error is expected for non-existent item")
}
}
}

for i := 10; i < 90; i++ {
blob, err := db.Ancient("a", uint64(i))
if err != nil {
t.Fatalf("Failed to retrieve item, %v", err)
}
if !bytes.Equal(blob, data[i]) {
t.Fatalf("Unexpected item content, want: %v, got: %v", data[i], blob)
}
}

_, err = db.Ancient("b", uint64(0))
if err == nil {
t.Fatal("Error is expected for unknown table")
}
}

func batchRead(t *testing.T, newFn func(kinds []string) ethdb.AncientStore) {
var (
db   = newFn([]string{"a"})
data = makeDataset(100, 32)
)
defer db.Close()

if _, err := db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
for i := 0; i < 100; i++ {
if err := op.AppendRaw("a", uint64(i), data[i]); err != nil {
return err
}
}
return nil
}); err != nil {
t.Fatalf("Failed to write ancient data %v", err)
}
db.TruncateTail(10)
db.TruncateHead(90)

var cases = []struct {
start    uint64
count    uint64
maxSize  uint64
expStart int
expLimit int
}{
{10, 80, 0, 10, 90},
{10, 80, 32, 10, 11},
{10, 80, 31, 10, 11},
{10, 80, 32 * 80, 10, 90},
{10, 90, 0, 10, 90},
}
for i, c := range cases {
batch, err := db.AncientRange("a", c.start, c.count, c.maxSize)
if err != nil {
t.Fatalf("Failed to retrieve item in range, %v", err)
}
if !reflect.DeepEqual(batch, data[c.expStart:c.expLimit]) {
t.Fatalf("Case %d, Batch content is not matched", i)
}
}

_, err := db.AncientRange("a", 0, 1, 0)
if err == nil {
t.Fatal("Out-of-range retrieval should be rejected")
}
_, err = db.AncientRange("a", 90, 1, 0)
if err == nil {
t.Fatal("Out-of-range retrieval should be rejected")
}
_, err = db.AncientRange("a", 10, 0, 0)
if err == nil {
t.Fatal("Zero-size retrieval should be rejected")
}
_, err = db.AncientRange("b", 10, 1, 0)
if err == nil {
t.Fatal("Item in unknown table shouldn'\''t be found")
}
}

func basicWrite(t *testing.T, newFn func(kinds []string) ethdb.AncientStore) {
var (
db    = newFn([]string{"a", "b"})
dataA = makeDataset(100, 32)
dataB = makeDataset(100, 32)
)
defer db.Close()

_, err := db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
for i := 0; i < 100; i++ {
if err := op.AppendRaw("a", uint64(i), dataA[i]); err != nil {
return err
}
}
return nil
})
if err == nil {
t.Fatal("Unaligned ancient write should be rejected")
}

size, err := db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
for i := 0; i < 100; i++ {
if err := op.AppendRaw("a", uint64(i), dataA[i]); err != nil {
return err
}
if err := op.AppendRaw("b", uint64(i), dataB[i]); err != nil {
return err
}
}
return nil
})
if err != nil {
t.Fatalf("Failed to write ancient data %v", err)
}
wantSize := int64(6400)
if size != wantSize {
t.Fatalf("Ancient write size is not expected, want: %d, got: %d", wantSize, size)
}

db.TruncateHead(90)
_, err = db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
for i := 90; i < 100; i++ {
if err := op.AppendRaw("a", uint64(i), dataA[i]); err != nil {
return err
}
if err := op.AppendRaw("b", uint64(i), dataB[i]); err != nil {
return err
}
}
return nil
})
if err != nil {
t.Fatalf("Failed to write ancient data %v", err)
}

db.TruncateHead(0)
_, err = db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
for i := 0; i < 100; i++ {
if err := op.AppendRaw("a", uint64(i), dataA[i]); err != nil {
return err
}
if err := op.AppendRaw("b", uint64(i), dataB[i]); err != nil {
return err
}
}
return nil
})
if err != nil {
t.Fatalf("Failed to write ancient data %v", err)
}

db.TruncateTail(200)
head, err := db.Ancients()
if err != nil {
t.Fatalf("Failed to retrieve head ancients %v", err)
}
tail, err := db.Tail()
if err != nil {
t.Fatalf("Failed to retrieve tail ancients %v", err)
}
if head != 200 || tail != 200 {
t.Fatalf("Ancient head and tail are not expected")
}
_, err = db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
offset := uint64(200)
for i := 0; i < 100; i++ {
if err := op.AppendRaw("a", offset+uint64(i), dataA[i]); err != nil {
return err
}
if err := op.AppendRaw("b", offset+uint64(i), dataB[i]); err != nil {
return err
}
}
return nil
})
if err != nil {
t.Fatalf("Failed to write ancient data %v", err)
}
head, err = db.Ancients()
if err != nil {
t.Fatalf("Failed to retrieve head ancients %v", err)
}
tail, err = db.Tail()
if err != nil {
t.Fatalf("Failed to retrieve tail ancients %v", err)
}
if head != 300 || tail != 200 {
t.Fatalf("Ancient head and tail are not expected")
}
}

func nonMutable(t *testing.T, newFn func(kinds []string) ethdb.AncientStore) {
db := newFn([]string{"a"})
defer db.Close()

if _, err := db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
data := make([]byte, 100)
if err := op.AppendRaw("a", uint64(0), data); err != nil {
return err
}
for i := range data {
data[i] = 0xff
}
return nil
}); err != nil {
t.Fatalf("Failed to write ancient data %v", err)
}
data, err := db.Ancient("a", uint64(0))
if err != nil {
t.Fatal(err)
}
for k, v := range data {
if v != 0 {
t.Fatalf("byte %d != 0: %x", k, v)
}
}
}

// TestResettableAncientSuite runs a suite of tests against a resettable ancient
// database implementation.
func TestResettableAncientSuite(t *testing.T, newFn func(kinds []string) ethdb.ResettableAncientStore) {
t.Run("Reset", func(t *testing.T) {
var (
db   = newFn([]string{"a"})
data = makeDataset(100, 32)
)
defer db.Close()

if _, err := db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
for i := 0; i < 100; i++ {
if err := op.AppendRaw("a", uint64(i), data[i]); err != nil {
return err
}
}
return nil
}); err != nil {
t.Fatalf("Failed to write ancient data %v", err)
}
db.TruncateTail(10)
db.TruncateHead(90)

db.Reset()
if _, err := db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
for i := 0; i < 100; i++ {
if err := op.AppendRaw("a", uint64(i), data[i]); err != nil {
return err
}
}
return nil
}); err != nil {
t.Fatalf("Failed to write ancient data %v", err)
}
})
}

func makeDataset(size, value int) [][]byte {
var vals [][]byte
for i := 0; i < size; i += 1 {
vals = append(vals, testrand.Bytes(value))
}
return vals
}
