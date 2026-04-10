package validator

import (
	"os"
	"path/filepath"
	"strings"

	"silachain/pkg/types"
)

type KeyFile struct {
	Address    types.Address `json:"address"`
	PublicKey  string        `json:"public_key"`
	PrivateKey string        `json:"private_key"`
}

func (k *KeyFile) Validate() error {
	if k == nil {
		return ErrNilKeyFile
	}
	if strings.TrimSpace(string(k.Address)) == "" {
		return ErrInvalidKeyFile
	}
	if strings.TrimSpace(k.PublicKey) == "" {
		return ErrInvalidKeyFile
	}
	if strings.TrimSpace(k.PrivateKey) == "" {
		return ErrEmptyPrivateKey
	}
	return nil
}

func SaveKeyFile(path string, file *KeyFile) error {
	if strings.TrimSpace(path) == "" {
		return ErrEmptyKeyPath
	}
	if err := file.Validate(); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data := []byte("{\n" +
		`  "address": "` + string(file.Address) + "\",\n" +
		`  "public_key": "` + file.PublicKey + "\",\n" +
		`  "private_key": "` + file.PrivateKey + "\"\n" +
		"}\n")

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}
