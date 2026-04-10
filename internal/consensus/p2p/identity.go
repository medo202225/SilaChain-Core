package p2p

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"

	chaincrypto "silachain/pkg/crypto"
)

type identityFile struct {
	PrivateKeyHex string `json:"private_key_hex"`
}

type Identity struct {
	ECDSAPrivateKey *ecdsa.PrivateKey
	PeerID          string
	PrivateKeyHex   string
	PublicKeyHex    string
}

func LoadOrCreateIdentity(path string) (*Identity, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create identity directory: %w", err)
	}

	if _, err := os.Stat(path); err == nil {
		return loadIdentity(path)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat identity file: %w", err)
	}

	nativePriv, _, err := chaincrypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate secp256k1 key: %w", err)
	}

	privHex := chaincrypto.PrivateKeyToHex(nativePriv)
	record := identityFile{PrivateKeyHex: privHex}

	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal identity file: %w", err)
	}

	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return nil, fmt.Errorf("write identity file: %w", err)
	}

	return buildIdentityFromHex(privHex)
}

func loadIdentity(path string) (*Identity, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read identity file: %w", err)
	}

	var record identityFile
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, fmt.Errorf("decode identity file: %w", err)
	}

	keyHex := strings.TrimSpace(record.PrivateKeyHex)
	if keyHex == "" {
		return nil, fmt.Errorf("identity file %s has empty private_key_hex", path)
	}

	return buildIdentityFromHex(keyHex)
}

func buildIdentityFromHex(privHex string) (*Identity, error) {
	nativePriv, err := chaincrypto.HexToPrivateKey(privHex)
	if err != nil {
		return nil, fmt.Errorf("parse private_key_hex as native secp256k1 key: %w", err)
	}

	peerID, err := DerivePeerIDFromPublicKey(&nativePriv.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("derive peer id: %w", err)
	}

	return &Identity{
		ECDSAPrivateKey: nativePriv,
		PeerID:          peerID,
		PrivateKeyHex:   chaincrypto.PrivateKeyToHex(nativePriv),
		PublicKeyHex:    chaincrypto.PublicKeyToHex(&nativePriv.PublicKey),
	}, nil
}

func DerivePeerIDFromPublicKey(pub *ecdsa.PublicKey) (string, error) {
	uncompressed := elliptic.Marshal(secp.S256(), pub.X, pub.Y)
	if len(uncompressed) == 0 {
		return "", fmt.Errorf("marshal public key")
	}

	parsed, err := secp.ParsePubKey(uncompressed)
	if err != nil {
		return "", fmt.Errorf("parse secp256k1 public key: %w", err)
	}

	compressed := parsed.SerializeCompressed()
	if len(compressed) != 33 {
		return "", fmt.Errorf("unexpected compressed public key length: %d", len(compressed))
	}

	protobuf, err := marshalLibp2pSecp256k1PublicKey(compressed)
	if err != nil {
		return "", err
	}

	multihash, err := identityMultihash(protobuf)
	if err != nil {
		return "", err
	}

	return base58Encode(multihash), nil
}

func marshalLibp2pSecp256k1PublicKey(compressedPubKey []byte) ([]byte, error) {
	if len(compressedPubKey) != 33 {
		return nil, fmt.Errorf("compressed secp256k1 public key must be 33 bytes, got %d", len(compressedPubKey))
	}

	out := make([]byte, 0, 2+2+33)
	out = append(out, 0x08)
	out = append(out, 0x02)
	out = append(out, 0x12)
	out = append(out, 0x21)
	out = append(out, compressedPubKey...)
	return out, nil
}

func identityMultihash(data []byte) ([]byte, error) {
	if len(data) > 127 {
		return nil, fmt.Errorf("identity multihash data too long for current implementation: %d", len(data))
	}

	out := make([]byte, 0, 2+len(data))
	out = append(out, 0x00)
	out = append(out, byte(len(data)))
	out = append(out, data...)
	return out, nil
}

func base58Encode(input []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	x := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)

	var encoded []byte
	for x.Cmp(zero) > 0 {
		x.DivMod(x, base, mod)
		encoded = append(encoded, alphabet[mod.Int64()])
	}

	for _, b := range input {
		if b == 0x00 {
			encoded = append(encoded, alphabet[0])
		} else {
			break
		}
	}

	for i, j := 0, len(encoded)-1; i < j; i, j = i+1, j-1 {
		encoded[i], encoded[j] = encoded[j], encoded[i]
	}

	if len(encoded) == 0 {
		return ""
	}

	return string(encoded)
}
