package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var ErrCorruptStorageFile = errors.New("corrupt storage file")

type DB struct {
	BasePath string
}

func NewDB(basePath string) *DB {
	if strings.TrimSpace(basePath) == "" {
		basePath = "data/node"
	}
	return &DB{BasePath: basePath}
}

func (db *DB) BaseDir() string {
	if db == nil {
		return ""
	}
	return db.BasePath
}

func (db *DB) Path(name string) string {
	if db == nil {
		return ""
	}
	return filepath.Join(db.BasePath, name)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmpPath := path + ".tmp"

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}

func (db *DB) CleanupTempFiles() error {
	if db == nil {
		return nil
	}
	if err := ensureDir(db.BasePath); err != nil {
		return err
	}

	entries, err := os.ReadDir(db.BasePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".tmp") {
			_ = os.Remove(filepath.Join(db.BasePath, entry.Name()))
		}
	}

	return nil
}

func (db *DB) WriteJSONAtomic(name string, v any) error {
	if db == nil {
		return nil
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(db.Path(name), data, 0o600)
}

func (db *DB) ReadJSON(name string, out any) error {
	if db == nil {
		return nil
	}

	if err := db.CleanupTempFiles(); err != nil {
		return err
	}

	data, err := os.ReadFile(db.Path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, out); err != nil {
		return ErrCorruptStorageFile
	}

	return nil
}
