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
"io"
"os"
"path/filepath"
)

func atomicRename(src, dest string) error {
if err := os.Rename(src, dest); err != nil {
return err
}
return syncDir(filepath.Dir(src))
}

// copyFrom copies data from 'srcPath' at offset 'offset' into 'destPath'.
func copyFrom(srcPath, destPath string, offset uint64, before func(f *os.File) error) error {
f, err := os.CreateTemp(filepath.Dir(destPath), "*")
if err != nil {
return err
}
fname := f.Name()

defer func() {
if f != nil {
f.Close()
}
os.Remove(fname)
}()
if before != nil {
if err := before(f); err != nil {
return err
}
}
src, err := os.Open(srcPath)
if err != nil {
return err
}
if _, err = src.Seek(int64(offset), 0); err != nil {
src.Close()
return err
}
_, err = io.Copy(f, src)
if err != nil {
src.Close()
return err
}
src.Close()

if err := f.Close(); err != nil {
return err
}
f = nil

return atomicRename(fname, destPath)
}

// reset atomically replaces the file at the given path with the provided content.
func reset(path string, content []byte) error {
f, err := os.CreateTemp(filepath.Dir(path), "*")
if err != nil {
return err
}
fname := f.Name()

defer func() {
if f != nil {
f.Close()
}
os.Remove(fname)
}()

_, err = f.Write(content)
if err != nil {
return err
}
if err := f.Sync(); err != nil {
return err
}
if err := f.Close(); err != nil {
return err
}
f = nil

return atomicRename(fname, path)
}

// openFreezerFileForAppend opens a freezer table file and seeks to the end.
func openFreezerFileForAppend(filename string) (*os.File, error) {
file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
if err != nil {
return nil, err
}
if _, err = file.Seek(0, io.SeekEnd); err != nil {
return nil, err
}
return file, nil
}

// openFreezerFileForReadOnly opens a freezer table file for read only access.
func openFreezerFileForReadOnly(filename string) (*os.File, error) {
return os.OpenFile(filename, os.O_RDONLY, 0644)
}

// openFreezerFileTruncated opens a freezer table making sure it is truncated.
func openFreezerFileTruncated(filename string) (*os.File, error) {
return os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
}

// truncateFreezerFile resizes a freezer table file and seeks to the end.
func truncateFreezerFile(file *os.File, size int64) error {
if err := file.Truncate(size); err != nil {
return err
}
if _, err := file.Seek(0, io.SeekEnd); err != nil {
return err
}
return nil
}

// grow prepares the slice space for new item, and doubles the slice capacity if needed.
func grow(buf []byte, n int) []byte {
if cap(buf)-len(buf) < n {
newcap := 2 * cap(buf)
if newcap-len(buf) < n {
newcap = len(buf) + n
}
nbuf := make([]byte, len(buf), newcap)
copy(nbuf, buf)
buf = nbuf
}
buf = buf[:len(buf)+n]
return buf
}

// syncDir flushes the directory to disk.
func syncDir(dir string) error {
d, err := os.Open(dir)
if err != nil {
return err
}
defer d.Close()
return d.Sync()
}
