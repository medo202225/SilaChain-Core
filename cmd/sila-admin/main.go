package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sila-admin <command>")
		fmt.Fprintln(os.Stderr, "commands: activate-node1")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "activate-node1":
		if err := activateNode("node1"); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println("node1 activated successfully")
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		os.Exit(1)
	}
}

func activateNode(nodeName string) error {
	base := filepath.Join("config", "mainnet")
	publicDir := filepath.Join(base, "public")
	nodeDir := filepath.Join(base, "nodes", nodeName)

	requiredPublic := []string{
		"protocol.json",
		"validators.json",
		"bootnodes.json",
		"peers.json",
	}

	for _, name := range requiredPublic {
		src := filepath.Join(publicDir, name)
		dst := filepath.Join(base, name)
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", name, err)
		}
	}

	genesisSrc := filepath.Join(publicDir, "genesis.json")
	genesisDst := filepath.Join(base, "genesis.json")
	if _, err := os.Stat(genesisSrc); err == nil {
		if err := copyFile(genesisSrc, genesisDst); err != nil {
			return fmt.Errorf("copy genesis.json: %w", err)
		}
	}

	if err := copyFile(filepath.Join(nodeDir, "node.json"), filepath.Join(base, "node.json")); err != nil {
		return fmt.Errorf("copy node.json: %w", err)
	}
	if err := copyFile(filepath.Join(nodeDir, "validator.key"), filepath.Join(base, "validator.key")); err != nil {
		return fmt.Errorf("copy validator.key: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("source missing: %s", src)
		}
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0o644)
}
