package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	chaincrypto "silachain/pkg/crypto"
)

type Output struct {
	ValidatorAddress string `json:"validator_address"`
	PublicKey        string `json:"public_key"`
	KeyPath          string `json:"key_path"`
}

func main() {
	var outPath string

	flag.StringVar(&outPath, "out", "validator.key", "path to save validator private key")
	flag.Parse()

	if err := run(outPath); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(outPath string) error {
	privateKey, publicKey, err := chaincrypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate key pair: %w", err)
	}

	address := chaincrypto.PublicKeyToAddress(publicKey)
	privateKeyHex := chaincrypto.PrivateKeyToHex(privateKey)
	publicKeyHex := chaincrypto.PublicKeyToHex(publicKey)

	dir := filepath.Dir(outPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create key directory: %w", err)
		}
	}

	if err := os.WriteFile(outPath, []byte(privateKeyHex), 0o600); err != nil {
		return fmt.Errorf("write validator key: %w", err)
	}

	out := Output{
		ValidatorAddress: string(address),
		PublicKey:        publicKeyHex,
		KeyPath:          outPath,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

var (
	_ func() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) = chaincrypto.GenerateKeyPair
)
