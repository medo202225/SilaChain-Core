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
"fmt"
"math/rand"
"os"
"path/filepath"
"testing"
"time"

"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/metrics"
)

var freezerTestTableDef = map[string]freezerTableConfig{
"test": {noSnappy: false, prunable: true},
}

func TestFreezerTableBasics(t *testing.T) {
t.Parallel()
// Create a temporary directory for the freezer
dir := t.TempDir()

// Open a new freezer table
table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table.Close()

// Write some items
items := []string{"item1", "item2", "item3", "item4", "item5"}
for i, item := range items {
if err := table.Append(uint64(i), []byte(item)); err != nil {
t.Fatalf("Failed to append item %d: %v", i, err)
}
}

// Read them back
for i, want := range items {
got, err := table.Retrieve(uint64(i))
if err != nil {
t.Fatalf("Failed to retrieve item %d: %v", i, err)
}
if !bytes.Equal(got, []byte(want)) {
t.Fatalf("Item %d mismatch: got %s, want %s", i, got, want)
}
}

// Check the count
if count := table.items.Load(); count != uint64(len(items)) {
t.Fatalf("Item count mismatch: got %d, want %d", count, len(items))
}
}

func TestFreezerTableAppendOutOfOrder(t *testing.T) {
t.Parallel()
dir := t.TempDir()

table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table.Close()

// Append item 0
if err := table.Append(0, []byte("item0")); err != nil {
t.Fatal(err)
}

// Try to append item 2 (skipping item 1)
if err := table.Append(2, []byte("item2")); err == nil {
t.Fatal("Expected error when appending out of order, got nil")
}
}

func TestFreezerTableRetrieveOutOfBounds(t *testing.T) {
t.Parallel()
dir := t.TempDir()

table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table.Close()

// Append one item
if err := table.Append(0, []byte("item0")); err != nil {
t.Fatal(err)
}

// Try to retrieve non-existent item
if _, err := table.Retrieve(1); err == nil {
t.Fatal("Expected error when retrieving out of bounds, got nil")
}
}

func TestFreezerTableTruncateHead(t *testing.T) {
t.Parallel()
dir := t.TempDir()

table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table.Close()

// Append 10 items
for i := 0; i < 10; i++ {
if err := table.Append(uint64(i), []byte(fmt.Sprintf("item%d", i))); err != nil {
t.Fatal(err)
}
}

// Truncate to 5 items
if err := table.truncateHead(5); err != nil {
t.Fatal(err)
}

// Verify only 5 items remain
if count := table.items.Load(); count != 5 {
t.Fatalf("Item count after truncate: got %d, want 5", count)
}

// Verify items 0-4 are still accessible
for i := 0; i < 5; i++ {
if _, err := table.Retrieve(uint64(i)); err != nil {
t.Fatalf("Item %d should be accessible: %v", i, err)
}
}

// Verify item 5 is not accessible
if _, err := table.Retrieve(5); err == nil {
t.Fatal("Item 5 should not be accessible")
}
}

func TestFreezerTableTruncateTail(t *testing.T) {
t.Parallel()
dir := t.TempDir()

table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table.Close()

// Append 10 items
for i := 0; i < 10; i++ {
if err := table.Append(uint64(i), []byte(fmt.Sprintf("item%d", i))); err != nil {
t.Fatal(err)
}
}

// Truncate tail to 5 (remove items 0-4)
if err := table.truncateTail(5); err != nil {
t.Fatal(err)
}

// Verify items 0-4 are not accessible
for i := 0; i < 5; i++ {
if _, err := table.Retrieve(uint64(i)); err == nil {
t.Fatalf("Item %d should not be accessible", i)
}
}

// Verify items 5-9 are accessible
for i := 5; i < 10; i++ {
if _, err := table.Retrieve(uint64(i)); err != nil {
t.Fatalf("Item %d should be accessible: %v", i, err)
}
}
}

func TestFreezerTableRepair(t *testing.T) {
t.Parallel()
dir := t.TempDir()

// Create and populate a table
table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
for i := 0; i < 100; i++ {
if err := table.Append(uint64(i), []byte(fmt.Sprintf("item%d", i))); err != nil {
t.Fatal(err)
}
}
table.Close()

// Reopen and verify repair works
table2, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table2.Close()

// Verify items are still accessible
for i := 0; i < 100; i++ {
got, err := table2.Retrieve(uint64(i))
if err != nil {
t.Fatalf("Failed to retrieve item %d after repair: %v", i, err)
}
want := []byte(fmt.Sprintf("item%d", i))
if !bytes.Equal(got, want) {
t.Fatalf("Item %d mismatch after repair: got %s, want %s", i, got, want)
}
}
}

func TestFreezerTableReadOnly(t *testing.T) {
t.Parallel()
dir := t.TempDir()

// Create and populate a table
table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
for i := 0; i < 10; i++ {
if err := table.Append(uint64(i), []byte(fmt.Sprintf("item%d", i))); err != nil {
t.Fatal(err)
}
}
table.Close()

// Open in read-only mode
roTable, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], true)
if err != nil {
t.Fatal(err)
}
defer roTable.Close()

// Verify reads work
for i := 0; i < 10; i++ {
if _, err := roTable.Retrieve(uint64(i)); err != nil {
t.Fatalf("Failed to retrieve item %d in read-only mode: %v", i, err)
}
}

// Verify writes fail
if err := roTable.Append(10, []byte("item10")); err == nil {
t.Fatal("Expected error when writing in read-only mode")
}
}

func TestFreezerTableMetrics(t *testing.T) {
t.Parallel()
dir := t.TempDir()

readMeter := metrics.NewInactiveMeter()
writeMeter := metrics.NewInactiveMeter()
sizeGauge := metrics.NewInactiveGauge()

table, err := newTable(dir, "test", &readMeter, &writeMeter, &sizeGauge, freezerTableSize, freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table.Close()

// Write some items
for i := 0; i < 10; i++ {
if err := table.Append(uint64(i), []byte(fmt.Sprintf("item%d", i))); err != nil {
t.Fatal(err)
}
}

// Read them back
for i := 0; i < 10; i++ {
if _, err := table.Retrieve(uint64(i)); err != nil {
t.Fatal(err)
}
}
}

func TestFreezerTableLargeData(t *testing.T) {
t.Parallel()
dir := t.TempDir()

table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table.Close()

// Create a large data item (1MB)
largeData := make([]byte, 1024*1024)
rand.Read(largeData)

// Write it
if err := table.Append(0, largeData); err != nil {
t.Fatal(err)
}

// Read it back
got, err := table.Retrieve(0)
if err != nil {
t.Fatal(err)
}
if !bytes.Equal(got, largeData) {
t.Fatal("Large data mismatch")
}
}

func TestFreezerTableMultipleFiles(t *testing.T) {
t.Parallel()
dir := t.TempDir()

// Create a table with small max file size to force multiple files
table, err := newTable(dir, "test", metrics.NewInactiveMeter(), metrics.NewInactiveMeter(), metrics.NewInactiveGauge(), 1024, freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table.Close()

// Write many items to create multiple files
for i := 0; i < 100; i++ {
data := bytes.Repeat([]byte{byte(i)}, 100)
if err := table.Append(uint64(i), data); err != nil {
t.Fatalf("Failed to append item %d: %v", i, err)
}
}

// Verify all items can be read
for i := 0; i < 100; i++ {
got, err := table.Retrieve(uint64(i))
if err != nil {
t.Fatalf("Failed to retrieve item %d: %v", i, err)
}
want := bytes.Repeat([]byte{byte(i)}, 100)
if !bytes.Equal(got, want) {
t.Fatalf("Item %d mismatch", i)
}
}

// Check that multiple files were created
files, err := filepath.Glob(filepath.Join(dir, "test.*.cdat"))
if err != nil {
t.Fatal(err)
}
if len(files) < 2 {
t.Fatalf("Expected multiple data files, got %d", len(files))
}
}

func TestFreezerTableConcurrent(t *testing.T) {
t.Parallel()
dir := t.TempDir()

table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table.Close()

// Concurrent writes
done := make(chan bool)
for i := 0; i < 10; i++ {
go func(id int) {
for j := 0; j < 10; j++ {
idx := uint64(id*10 + j)
data := []byte(fmt.Sprintf("item%d", idx))
if err := table.Append(idx, data); err != nil {
t.Errorf("Failed to append item %d: %v", idx, err)
}
}
done <- true
}(i)
}

// Wait for all writes to complete
for i := 0; i < 10; i++ {
<-done
}

// Concurrent reads
for i := 0; i < 10; i++ {
go func(id int) {
for j := 0; j < 10; j++ {
idx := uint64(id*10 + j)
want := []byte(fmt.Sprintf("item%d", idx))
got, err := table.Retrieve(idx)
if err != nil {
t.Errorf("Failed to retrieve item %d: %v", idx, err)
}
if !bytes.Equal(got, want) {
t.Errorf("Item %d mismatch", idx)
}
}
done <- true
}(i)
}

for i := 0; i < 10; i++ {
<-done
}
}

func TestFreezerTableReset(t *testing.T) {
t.Parallel()
dir := t.TempDir()

table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table.Close()

// Append 10 items
for i := 0; i < 10; i++ {
if err := table.Append(uint64(i), []byte(fmt.Sprintf("item%d", i))); err != nil {
t.Fatal(err)
}
}

// Reset to tail 20
if err := table.resetTo(20); err != nil {
t.Fatal(err)
}

// Verify items 0-9 are not accessible
for i := 0; i < 10; i++ {
if _, err := table.Retrieve(uint64(i)); err == nil {
t.Fatalf("Item %d should not be accessible after reset", i)
}
}

// Verify new items can be appended starting from 20
if err := table.Append(20, []byte("item20")); err != nil {
t.Fatal(err)
}

got, err := table.Retrieve(20)
if err != nil {
t.Fatal(err)
}
if !bytes.Equal(got, []byte("item20")) {
t.Fatal("Item 20 mismatch")
}
}

func TestFreezerTableSync(t *testing.T) {
t.Parallel()
dir := t.TempDir()

table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
t.Fatal(err)
}
defer table.Close()

// Append some items without sync
for i := 0; i < 10; i++ {
if err := table.Append(uint64(i), []byte(fmt.Sprintf("item%d", i))); err != nil {
t.Fatal(err)
}
}

// Force sync
if err := table.Sync(); err != nil {
t.Fatal(err)
}

// Verify items are still accessible
for i := 0; i < 10; i++ {
if _, err := table.Retrieve(uint64(i)); err != nil {
t.Fatalf("Item %d not accessible after sync: %v", i, err)
}
}
}

// Benchmarks
func BenchmarkFreezerTableAppend(b *testing.B) {
dir := b.TempDir()
table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
b.Fatal(err)
}
defer table.Close()

data := bytes.Repeat([]byte{0x42}, 1024) // 1KB item
b.ResetTimer()
b.ReportAllocs()

for i := 0; i < b.N; i++ {
if err := table.Append(uint64(i), data); err != nil {
b.Fatal(err)
}
}
}

func BenchmarkFreezerTableRetrieve(b *testing.B) {
dir := b.TempDir()
table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
b.Fatal(err)
}
defer table.Close()

data := bytes.Repeat([]byte{0x42}, 1024) // 1KB item
for i := 0; i < 10000; i++ {
if err := table.Append(uint64(i), data); err != nil {
b.Fatal(err)
}
}

b.ResetTimer()
b.ReportAllocs()

for i := 0; i < b.N; i++ {
idx := uint64(rand.Intn(10000))
if _, err := table.Retrieve(idx); err != nil {
b.Fatal(err)
}
}
}

func BenchmarkFreezerTableRetrieveItems(b *testing.B) {
dir := b.TempDir()
table, err := newFreezerTable(dir, "test", freezerTestTableDef["test"], false)
if err != nil {
b.Fatal(err)
}
defer table.Close()

data := bytes.Repeat([]byte{0x42}, 1024) // 1KB item
for i := 0; i < 10000; i++ {
if err := table.Append(uint64(i), data); err != nil {
b.Fatal(err)
}
}

b.ResetTimer()
b.ReportAllocs()

for i := 0; i < b.N; i++ {
start := uint64(rand.Intn(9900))
if _, err := table.RetrieveItems(start, 100, 0); err != nil {
b.Fatal(err)
}
}
}
