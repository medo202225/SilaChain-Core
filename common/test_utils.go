// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadTestJSON reads a JSON file and unmarshals it into the provided target.
func LoadTestJSON(file string, target interface{}) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(content, target); err != nil {
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			line := findJSONLine(content, syntaxErr.Offset)
			return fmt.Errorf("JSON syntax error at %v:%v: %v", file, line, err)
		}
		return fmt.Errorf("JSON unmarshal error in %v: %v", file, err)
	}

	return nil
}

// findJSONLine returns the line number for the given byte offset into data.
func findJSONLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
}
