// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// PathExists reports whether a filesystem entry exists at the given path.
func PathExists(targetPath string) bool {
	_, err := os.Stat(targetPath)
	return !errors.Is(err, fs.ErrNotExist)
}

// ResolveAbsolutePath returns baseDir + relativeName, or relativeName if it is already absolute.
func ResolveAbsolutePath(baseDir string, relativeName string) string {
	if filepath.IsAbs(relativeName) {
		return relativeName
	}
	return filepath.Join(baseDir, relativeName)
}

// IsDirectoryNonEmpty reports whether a directory exists and contains at least one entry.
func IsDirectoryNonEmpty(dir string) bool {
	handle, err := os.Open(dir)
	if err != nil {
		return false
	}
	defer handle.Close()

	names, _ := handle.Readdirnames(1)
	return len(names) > 0
}
