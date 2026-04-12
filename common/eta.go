// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import "time"

// EstimateRemainingDuration estimates the remaining time for unfinished work
// from completed units, remaining units, and elapsed processing time.
func EstimateRemainingDuration(doneUnits, remainingUnits uint64, elapsed time.Duration) time.Duration {
	if doneUnits == 0 || elapsed <= 0 {
		return 0
	}

	elapsedMillis := elapsed.Milliseconds()
	if elapsedMillis == 0 {
		return 0
	}

	unitsPerMillisecond := float64(doneUnits) / float64(elapsedMillis)
	if unitsPerMillisecond <= 0 {
		return 0
	}

	return time.Duration(float64(remainingUnits)/unitsPerMillisecond) * time.Millisecond
}
