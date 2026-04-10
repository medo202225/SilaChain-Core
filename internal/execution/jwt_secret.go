package execution

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func EnsureJWTSecretFile(path string) (string, error) {
	clean := filepath.Clean(path)
	if strings.TrimSpace(clean) == "" {
		return "", fmt.Errorf("engine jwt secret path is empty")
	}

	if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
		return "", fmt.Errorf("create jwt secret dir: %w", err)
	}

	if raw, err := os.ReadFile(clean); err == nil {
		secret := strings.TrimSpace(string(raw))
		if len(secret) != 64 {
			return "", fmt.Errorf("engine jwt secret must be 32 bytes hex, got length %d", len(secret))
		}
		if _, err := hex.DecodeString(secret); err != nil {
			return "", fmt.Errorf("engine jwt secret decode failed: %w", err)
		}
		return secret, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read engine jwt secret failed: %w", err)
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate engine jwt secret failed: %w", err)
	}

	secret := hex.EncodeToString(buf)
	if err := os.WriteFile(clean, []byte(secret), 0o600); err != nil {
		return "", fmt.Errorf("write engine jwt secret failed: %w", err)
	}

	return secret, nil
}
