package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	validatorclient "silachain/internal/validatorclient"
)

type Output struct {
	PublicKeyHex string `json:"public_key"`
	Path         string `json:"path"`
	KeystorePath string `json:"keystore_path"`
	SecretPath   string `json:"secret_path"`
}

func randomPassphrase() ([]byte, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	out := base64.RawStdEncoding.EncodeToString(b)
	return []byte(out), nil
}

func main() {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	var out string
	var secretOut string
	var path string

	fs.StringVar(&out, "out", "runtime/validators/node1/keystores/voting-keystore.json", "EIP-2335 keystore output path")
	fs.StringVar(&secretOut, "secret-out", "runtime/validators/node1/secrets/voting-keystore.pass", "keystore passphrase output path")
	fs.StringVar(&path, "path", "m/12381/3600/0/0/0", "ERC-2334 derivation path")
	_ = fs.Parse(os.Args[1:])

	passphrase, err := randomPassphrase()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	res, err := validatorclient.CreateVotingKeystore(out, secretOut, path, passphrase)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(Output{
		PublicKeyHex: res.PublicKeyHex,
		Path:         res.Path,
		KeystorePath: res.KeystorePath,
		SecretPath:   res.SecretPath,
	})
}
