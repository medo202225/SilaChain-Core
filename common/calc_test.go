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
"testing"
"time"
)

func TestCalculateETA(t *testing.T) {
type args struct {
done    uint64
left    uint64
elapsed time.Duration
}
tests := []struct {
name string
args args
want time.Duration
}{
{
name: "zero done",
args: args{done: 0, left: 100, elapsed: time.Second},
want: 0,
},
{
name: "zero elapsed",
args: args{done: 1, left: 100, elapsed: 0},
want: 0,
},
{
name: "half done, half left",
args: args{done: 10, left: 10, elapsed: time.Second},
want: time.Second,
},
{
name: "one third done, two thirds left",
args: args{done: 10, left: 20, elapsed: time.Second},
want: 2 * time.Second,
},
{
name: "small numbers",
args: args{done: 1, left: 99, elapsed: 10 * time.Millisecond},
want: 990 * time.Millisecond,
},
{
name: "@Jolly23 's case",
args: args{done: 16858580, left: 41802252, elapsed: 66179848 * time.Millisecond},
want: 164098440 * time.Millisecond,
// wrong msg: msg="Indexing state history" processed=16858580 left=41802252 elapsed=18h22m59.848s eta=11h36m42.252s
// should be around 45.58 hours
},
}
for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
if got := CalculateETA(tt.args.done, tt.args.left, tt.args.elapsed); got != tt.want {
t.Errorf("CalculateETA() = %v ms, want %v ms", got.Milliseconds(), tt.want.Milliseconds())
}
})
}
}

func BenchmarkCalculateETA(b *testing.B) {
for i := 0; i < b.N; i++ {
CalculateETA(1000, 5000, 10*time.Second)
}
}
