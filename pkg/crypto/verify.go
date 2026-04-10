package crypto

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

func RecoverPublicKey(hash []byte, sig []byte) (*ecdsa.PublicKey, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("invalid hash length: got %d want 32", len(hash))
	}
	if len(sig) != 65 {
		return nil, fmt.Errorf("invalid signature length: got %d want 65", len(sig))
	}

	v := sig[64]
	if v > 3 {
		return nil, fmt.Errorf("invalid recovery id: %d", v)
	}

	compact := make([]byte, 65)
	compact[0] = 27 + v
	copy(compact[1:], sig[:64])

	pub, _, err := secpecdsa.RecoverCompact(compact, hash)
	if err != nil {
		return nil, fmt.Errorf("recover public key: %w", err)
	}

	ecdsaPub := pub.ToECDSA()
	return ecdsaPub, nil
}

func Verify(hash []byte, sig []byte, pub *ecdsa.PublicKey) bool {
	if pub == nil || len(hash) != 32 || len(sig) != 65 {
		return false
	}

	recovered, err := RecoverPublicKey(hash, sig)
	if err != nil {
		return false
	}

	return PublicKeyToHex(recovered) == PublicKeyToHex(pub)
}

func VerifyHashHex(pub *ecdsa.PublicKey, hash string, sigHex string) (bool, error) {
	rawHash, err := DecodeHexString(hash)
	if err != nil {
		return false, fmt.Errorf("decode hash hex: %w", err)
	}

	rawSig, err := DecodeHexString(sigHex)
	if err != nil {
		return false, fmt.Errorf("decode signature hex: %w", err)
	}

	return Verify(rawHash, rawSig, pub), nil
}

func ValidateAddressMatchesPublicKey(address string, publicKeyHex string) error {
	pub, err := HexToPublicKey(publicKeyHex)
	if err != nil {
		return err
	}

	derived := PublicKeyToAddress(pub)
	if strings.TrimSpace(string(derived)) != strings.TrimSpace(address) {
		return fmt.Errorf("address/public key mismatch")
	}

	return nil
}

func VerifySignatureHexByPublicKeyHex(publicKeyHex string, hashHex string, signatureHex string) (bool, error) {
	pub, err := HexToPublicKey(publicKeyHex)
	if err != nil {
		return false, err
	}

	return VerifyHashHex(pub, hashHex, signatureHex)
}
