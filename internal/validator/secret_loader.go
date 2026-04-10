package validator

import (
	"fmt"
	"os"
	"strings"
)

func LoadSecretFile(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("validator secret path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	secret := strings.TrimSpace(string(data))
	if secret == "" {
		return "", fmt.Errorf("validator secret file is empty")
	}

	return secret, nil
}
