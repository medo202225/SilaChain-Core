// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import (
	"testing"
	"time"
)

func TestEstimateRemainingDuration(t *testing.T) {
	type args struct {
		doneUnits      uint64
		remainingUnits uint64
		elapsed        time.Duration
	}

	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "zero done units",
			args: args{
				doneUnits:      0,
				remainingUnits: 100,
				elapsed:        time.Second,
			},
			want: 0,
		},
		{
			name: "zero elapsed duration",
			args: args{
				doneUnits:      1,
				remainingUnits: 100,
				elapsed:        0,
			},
			want: 0,
		},
		{
			name: "large progress sample",
			args: args{
				doneUnits:      16858580,
				remainingUnits: 41802252,
				elapsed:        66179848 * time.Millisecond,
			},
			want: 164098440 * time.Millisecond,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := EstimateRemainingDuration(
				test.args.doneUnits,
				test.args.remainingUnits,
				test.args.elapsed,
			)
			if got != test.want {
				t.Errorf(
					"EstimateRemainingDuration() = %v ms, want %v ms",
					got.Milliseconds(),
					test.want.Milliseconds(),
				)
			}
		})
	}
}
