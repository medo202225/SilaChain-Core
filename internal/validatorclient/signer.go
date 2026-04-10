package validatorclient

// CANONICAL OWNERSHIP: validator client package for keystores, duties, slashing protection, signing, and validator service runtime.
// Planned final architectural name is validatorclient after dependency cleanup.

import (
	"context"
	"encoding/hex"
	"fmt"

	"silachain/internal/validatorclient/slashing"

	blsu "github.com/protolambda/bls12-381-util"
)

type SignatureResult struct {
	SignatureHex string `json:"signature_hex"`
}

func decodeValidatorPubKeyHex(publicKeyHex string) ([]byte, error) {
	if publicKeyHex == "" {
		return nil, fmt.Errorf("public key hex is empty")
	}
	pubRaw, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode public key hex: %w", err)
	}
	if len(pubRaw) != 48 {
		return nil, fmt.Errorf("unexpected public key length: %d", len(pubRaw))
	}
	return pubRaw, nil
}

func SignMessage(loaded *LoadedVotingKeystore, message []byte) (*SignatureResult, error) {
	if loaded == nil {
		return nil, fmt.Errorf("loaded voting keystore is nil")
	}
	if loaded.SecretKey == nil {
		return nil, fmt.Errorf("loaded voting secret key is nil")
	}
	if len(message) == 0 {
		return nil, fmt.Errorf("message is empty")
	}

	sig := blsu.Sign(loaded.SecretKey, message)
	sigBytes := sig.Serialize()

	return &SignatureResult{
		SignatureHex: hex.EncodeToString(sigBytes[:]),
	}, nil
}

func ProtectedSignBlock(
	ctx context.Context,
	loaded *LoadedVotingKeystore,
	store slashing.Store,
	slot uint64,
	signingRoot []byte,
) (*SignatureResult, error) {
	if loaded == nil {
		return nil, fmt.Errorf("loaded voting keystore is nil")
	}
	if loaded.SecretKey == nil {
		return nil, fmt.Errorf("loaded voting secret key is nil")
	}
	if store == nil {
		return nil, fmt.Errorf("slashing store is nil")
	}
	if len(signingRoot) == 0 {
		return nil, fmt.Errorf("signing root is empty")
	}

	pubKey, err := decodeValidatorPubKeyHex(loaded.PublicHex)
	if err != nil {
		return nil, err
	}

	if err := store.CheckAndRecordBlock(ctx, pubKey, slot, signingRoot); err != nil {
		return nil, err
	}

	sig := blsu.Sign(loaded.SecretKey, signingRoot)
	sigBytes := sig.Serialize()

	return &SignatureResult{
		SignatureHex: hex.EncodeToString(sigBytes[:]),
	}, nil
}

func ProtectedSignAttestation(
	ctx context.Context,
	loaded *LoadedVotingKeystore,
	store slashing.Store,
	sourceEpoch uint64,
	targetEpoch uint64,
	signingRoot []byte,
) (*SignatureResult, error) {
	if loaded == nil {
		return nil, fmt.Errorf("loaded voting keystore is nil")
	}
	if loaded.SecretKey == nil {
		return nil, fmt.Errorf("loaded voting secret key is nil")
	}
	if store == nil {
		return nil, fmt.Errorf("slashing store is nil")
	}
	if len(signingRoot) == 0 {
		return nil, fmt.Errorf("signing root is empty")
	}

	pubKey, err := decodeValidatorPubKeyHex(loaded.PublicHex)
	if err != nil {
		return nil, err
	}

	if err := store.CheckAndRecordAttestation(ctx, pubKey, sourceEpoch, targetEpoch, signingRoot); err != nil {
		return nil, err
	}

	sig := blsu.Sign(loaded.SecretKey, signingRoot)
	sigBytes := sig.Serialize()

	return &SignatureResult{
		SignatureHex: hex.EncodeToString(sigBytes[:]),
	}, nil
}

func VerifySignatureHex(publicKeyHex string, message []byte, signatureHex string) (bool, error) {
	if publicKeyHex == "" {
		return false, fmt.Errorf("public key hex is empty")
	}
	if signatureHex == "" {
		return false, fmt.Errorf("signature hex is empty")
	}
	if len(message) == 0 {
		return false, fmt.Errorf("message is empty")
	}

	pubRaw, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false, fmt.Errorf("decode public key hex: %w", err)
	}
	if len(pubRaw) != 48 {
		return false, fmt.Errorf("unexpected public key length: %d", len(pubRaw))
	}

	sigRaw, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, fmt.Errorf("decode signature hex: %w", err)
	}
	if len(sigRaw) != 96 {
		return false, fmt.Errorf("unexpected signature length: %d", len(sigRaw))
	}

	var pubBytes [48]byte
	copy(pubBytes[:], pubRaw)

	var sigBytes [96]byte
	copy(sigBytes[:], sigRaw)

	var pub blsu.Pubkey
	if err := pub.Deserialize(&pubBytes); err != nil {
		return false, fmt.Errorf("deserialize public key: %w", err)
	}

	var sig blsu.Signature
	if err := sig.Deserialize(&sigBytes); err != nil {
		return false, fmt.Errorf("deserialize signature: %w", err)
	}

	ok := blsu.Verify(&pub, message, &sig)
	return ok, nil
}
