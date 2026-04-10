package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func requireFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required file missing: %s", path)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but found directory: %s", path)
	}
	return nil
}

func ensureDirSafe(path string) error {
	if path == "" {
		return fmt.Errorf("empty directory path")
	}
	return os.MkdirAll(path, 0o755)
}

func runStartupChecks(configPath string, dataDir string) error {
	requiredFiles := []string{
		configPath,
		"config/mainnet/validators.json",
		"config/mainnet/genesis.json",
		"config/mainnet/protocol.json",
	}

	for _, p := range requiredFiles {
		if err := requireFile(p); err != nil {
			return err
		}
	}

	if err := ensureDirSafe(dataDir); err != nil {
		return err
	}

	validatorDirs := []string{
		filepath.Join("data", "validator"),
		filepath.Join("data", "validator", "consensus"),
		filepath.Join("data", "validator", "protection"),
	}

	for _, d := range validatorDirs {
		if err := ensureDirSafe(d); err != nil {
			return err
		}
	}

	return nil
}
