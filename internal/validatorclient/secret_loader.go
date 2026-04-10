package validatorclient

import (
	"fmt"
	"os"
	"strings"
)

func LoadSecretFile(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("secret path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	secret := strings.TrimSpace(string(data))
	if secret == "" {
		return nil, fmt.Errorf("secret file is empty")
	}

	return []byte(secret), nil
}
