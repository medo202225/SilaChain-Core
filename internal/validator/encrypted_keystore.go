package validator

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type EncryptedKeyFile struct {
	Address    string `json:"address"`
	PublicKey  string `json:"public_key"`
	Cipher     string `json:"cipher"`
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
	KDF        string `json:"kdf"`
	KDFSalt    string `json:"kdf_salt"`
	KDFRounds  int    `json:"kdf_rounds"`
	Version    int    `json:"version"`
}

func deriveKey(password string, salt []byte, rounds int) []byte {
	data := append([]byte(password), salt...)
	sum := sha256.Sum256(data)
	key := sum[:]
	for i := 1; i < rounds; i++ {
		next := sha256.Sum256(append(key, salt...))
		key = next[:]
	}
	return key
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

func SaveEncryptedKeyFile(path string, file *KeyFile, password string) error {
	if strings.TrimSpace(path) == "" {
		return ErrEmptyKeyPath
	}
	if file == nil {
		return ErrNilKeyFile
	}
	if err := file.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("validator keystore password is empty")
	}

	salt, err := randomBytes(16)
	if err != nil {
		return err
	}
	nonce, err := randomBytes(12)
	if err != nil {
		return err
	}

	rounds := 120000
	key := deriveKey(password, salt, rounds)

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(file.PrivateKey), nil)

	enc := EncryptedKeyFile{
		Address:    string(file.Address),
		PublicKey:  file.PublicKey,
		Cipher:     "AES-256-GCM",
		Ciphertext: hex.EncodeToString(ciphertext),
		Nonce:      hex.EncodeToString(nonce),
		KDF:        "SHA256-ITER",
		KDFSalt:    hex.EncodeToString(salt),
		KDFRounds:  rounds,
		Version:    1,
	}

	data, err := json.MarshalIndent(enc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

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

func LoadEncryptedKeyFile(path string, password string) (*KeyFile, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrEmptyKeyPath
	}
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("validator keystore password is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrKeyFileNotFound
		}
		return nil, err
	}

	var enc EncryptedKeyFile
	if err := json.Unmarshal(data, &enc); err != nil {
		return nil, ErrInvalidKeyFile
	}

	salt, err := hex.DecodeString(enc.KDFSalt)
	if err != nil {
		return nil, ErrInvalidKeyFile
	}
	nonce, err := hex.DecodeString(enc.Nonce)
	if err != nil {
		return nil, ErrInvalidKeyFile
	}
	ciphertext, err := hex.DecodeString(enc.Ciphertext)
	if err != nil {
		return nil, ErrInvalidKeyFile
	}

	key := deriveKey(password, salt, enc.KDFRounds)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidKeyFile
	}

	file := &KeyFile{
		Address:    AddressMust(enc.Address),
		PublicKey:  enc.PublicKey,
		PrivateKey: string(plain),
	}
	if err := file.Validate(); err != nil {
		return nil, err
	}
	return file, nil
}
