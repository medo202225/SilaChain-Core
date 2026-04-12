// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
)

// ReportDiagnostic prints a diagnostic error report to stderr with caller details and a stack trace.
func ReportDiagnostic(extra ...interface{}) {
	fmt.Fprintln(os.Stderr, "SilaChain encountered a diagnostic fault that should be reported to the maintainers.")
	if len(extra) > 0 {
		fmt.Fprintln(os.Stderr, extra...)
	}

	_, file, line, ok := runtime.Caller(1)
	if ok {
		fmt.Fprintf(os.Stderr, "%s:%d\n", file, line)
	}

	debug.PrintStack()

	fmt.Fprintln(os.Stderr, "#### SILACHAIN DIAGNOSTIC REPORT ####")
}

// PrintObsolescenceWarning prints a warning message inside a text box.
func PrintObsolescenceWarning(message string) {
	border := strings.Repeat("#", len(message)+4)
	padding := strings.Repeat(" ", len(message))

	fmt.Printf(
		"\n%s\n# %s #\n# %s #\n# %s #\n%s\n\n",
		border,
		padding,
		message,
		padding,
		border,
	)
}
