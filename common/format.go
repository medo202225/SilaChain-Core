// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// SilaPrettyDuration formats time.Duration values in a concise readable form.
type SilaPrettyDuration time.Duration

var silaPrettyDurationFractionRe = regexp.MustCompile(`\.[0-9]{4,}`)

// String returns the duration string with fractional precision trimmed to three decimals.
func (d SilaPrettyDuration) String() string {
	label := time.Duration(d).String()
	match := silaPrettyDurationFractionRe.FindString(label)
	if len(match) > 4 {
		label = strings.Replace(label, match, match[:4], 1)
	}
	return label
}

// SilaPrettyAge formats elapsed time from a timestamp using compact age units.
type SilaPrettyAge time.Time

var silaPrettyAgeUnits = []struct {
	Size   time.Duration
	Symbol string
}{
	{12 * 30 * 24 * time.Hour, "y"},
	{30 * 24 * time.Hour, "mo"},
	{7 * 24 * time.Hour, "w"},
	{24 * time.Hour, "d"},
	{time.Hour, "h"},
	{time.Minute, "m"},
	{time.Second, "s"},
}

// String returns a compact textual age using up to three significant units.
func (t SilaPrettyAge) String() string {
	diff := time.Since(time.Time(t))
	if diff < time.Second {
		return "0"
	}

	result := ""
	precision := 0

	for _, unit := range silaPrettyAgeUnits {
		if diff >= unit.Size {
			result = fmt.Sprintf("%s%d%s", result, diff/unit.Size, unit.Symbol)
			diff %= unit.Size
			precision++
			if precision >= 3 {
				break
			}
		}
	}

	return result
}
