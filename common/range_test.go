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

package common

import (
"slices"
"testing"
)

func TestRangeIter(t *testing.T) {
r := NewRange[uint32](1, 7)
values := slices.Collect(r.Iter())
if !slices.Equal(values, []uint32{1, 2, 3, 4, 5, 6, 7}) {
t.Fatalf("wrong iter values: %v", values)
}

empty := NewRange[uint32](1, 0)
values = slices.Collect(empty.Iter())
if !slices.Equal(values, []uint32{}) {
t.Fatalf("wrong iter values: %v", values)
}
}

func TestRangeBasics(t *testing.T) {
r := NewRange[uint32](5, 10)

// Test First and Count
if r.First() != 5 {
t.Errorf("First() = %d, want 5", r.First())
}
if r.Count() != 10 {
t.Errorf("Count() = %d, want 10", r.Count())
}

// Test Last
if r.Last() != 14 {
t.Errorf("Last() = %d, want 14", r.Last())
}

// Test AfterLast
if r.AfterLast() != 15 {
t.Errorf("AfterLast() = %d, want 15", r.AfterLast())
}

// Test IsEmpty
if r.IsEmpty() {
t.Error("IsEmpty() = true, want false")
}

empty := NewRange[uint32](5, 0)
if !empty.IsEmpty() {
t.Error("IsEmpty() = false, want true")
}
}

func TestRangeIncludes(t *testing.T) {
r := NewRange[uint32](10, 10) // 10 to 19

tests := []struct {
v    uint32
want bool
}{
{9, false},
{10, true},
{15, true},
{19, true},
{20, false},
}

for _, tt := range tests {
if got := r.Includes(tt.v); got != tt.want {
t.Errorf("Includes(%d) = %v, want %v", tt.v, got, tt.want)
}
}
}

func TestRangeSetters(t *testing.T) {
r := NewRange[uint32](10, 10)

// Test SetFirst
r.SetFirst(20)
if r.First() != 20 || r.AfterLast() != 20 {
t.Errorf("SetFirst(20): first=%d, afterLast=%d", r.First(), r.AfterLast())
}

// Test SetAfterLast
r.SetAfterLast(5)
if r.First() != 5 || r.AfterLast() != 5 {
t.Errorf("SetAfterLast(5): first=%d, afterLast=%d", r.First(), r.AfterLast())
}

// Test SetLast
r.SetLast(15)
if r.First() != 5 || r.AfterLast() != 16 {
t.Errorf("SetLast(15): first=%d, afterLast=%d", r.First(), r.AfterLast())
}
}

func TestRangeLastPanic(t *testing.T) {
defer func() {
if r := recover(); r == nil {
t.Error("Last() on empty range should panic")
}
}()

empty := NewRange[uint32](5, 0)
_ = empty.Last() // should panic
}

func TestRangeIntersection(t *testing.T) {
r1 := NewRange[uint32](10, 20) // 10 to 29
r2 := NewRange[uint32](25, 15) // 25 to 39

inter := r1.Intersection(r2)

if inter.First() != 25 || inter.AfterLast() != 30 {
t.Errorf("Intersection: first=%d, afterLast=%d, want first=25, afterLast=30", 
inter.First(), inter.AfterLast())
}

// Non-overlapping
r3 := NewRange[uint32](50, 10) // 50 to 59
empty := r1.Intersection(r3)

if !empty.IsEmpty() {
t.Error("Intersection of non-overlapping ranges should be empty")
}
}

func TestRangeUnion(t *testing.T) {
r1 := NewRange[uint32](10, 20) // 10 to 29
r2 := NewRange[uint32](30, 10) // 30 to 39

union := r1.Union(r2)

if union.First() != 10 || union.AfterLast() != 40 {
t.Errorf("Union: first=%d, afterLast=%d, want first=10, afterLast=40",
union.First(), union.AfterLast())
}
}

func TestRangeUnionPanic(t *testing.T) {
defer func() {
if r := recover(); r == nil {
t.Error("Union() on gapped ranges should panic")
}
}()

r1 := NewRange[uint32](10, 10) // 10 to 19
r2 := NewRange[uint32](25, 10) // 25 to 34

_ = r1.Union(r2) // should panic (gap between 19 and 25)
}

func TestRangeWithUint64(t *testing.T) {
r := NewRange[uint64](1000000000, 5)

if r.First() != 1000000000 {
t.Errorf("First() = %d, want 1000000000", r.First())
}
if r.Count() != 5 {
t.Errorf("Count() = %d, want 5", r.Count())
}

values := slices.Collect(r.Iter())
expected := []uint64{1000000000, 1000000001, 1000000002, 1000000003, 1000000004}
if !slices.Equal(values, expected) {
t.Fatalf("wrong iter values: %v", values)
}
}

func BenchmarkRangeIter(b *testing.B) {
r := NewRange[uint32](1, 1000)
b.ResetTimer()
for i := 0; i < b.N; i++ {
for v := range r.Iter() {
_ = v
}
}
}

func BenchmarkRangeIncludes(b *testing.B) {
r := NewRange[uint32](1, 1000)
b.ResetTimer()
for i := 0; i < b.N; i++ {
r.Includes(uint32(i % 1500))
}
}
