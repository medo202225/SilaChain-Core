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
		"config/networks/mainnet/public/validators.json",
		"config/networks/mainnet/public/genesis.json",
		"config/networks/mainnet/public/protocol.json",
	}

	for _, p := range requiredFiles {
		if err := requireFile(p); err != nil {
			return err
		}
	}

	if err := ensureDirSafe(dataDir); err != nil {
		return err
	}

	executionDirs := []string{
		dataDir,
		filepath.Join("runtime", "execution"),
		filepath.Join("runtime", "execution", "chain"),
		filepath.Join("runtime", "execution", "state"),
	}

	for _, d := range executionDirs {
		if err := ensureDirSafe(d); err != nil {
			return err
		}
	}

	return nil
}
