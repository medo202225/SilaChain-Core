package consensuslegacy

import (
	"fmt"
	"time"

	"silachain/internal/validator"
	chaincrypto "silachain/pkg/crypto"
	"silachain/pkg/types"
)

type Attestation struct {
	Slot        Slot          `json:"slot"`
	Epoch       Epoch         `json:"epoch"`
	Validator   types.Address `json:"validator"`
	PublicKey   string        `json:"public_key"`
	BlockHash   string        `json:"block_hash"`
	SourceEpoch Epoch         `json:"source_epoch"`
	TargetEpoch Epoch         `json:"target_epoch"`
	Timestamp   time.Time     `json:"timestamp"`
	Signature   string        `json:"signature"`
}

type attestationSigningPayload struct {
	Slot        Slot          `json:"slot"`
	Epoch       Epoch         `json:"epoch"`
	Validator   types.Address `json:"validator"`
	PublicKey   string        `json:"public_key"`
	BlockHash   string        `json:"block_hash"`
	SourceEpoch Epoch         `json:"source_epoch"`
	TargetEpoch Epoch         `json:"target_epoch"`
	Timestamp   int64         `json:"timestamp_unix"`
}

func NewAttestation(
	slot Slot,
	epoch Epoch,
	validatorAddress types.Address,
	publicKey string,
	blockHash string,
	sourceEpoch Epoch,
	targetEpoch Epoch,
	now time.Time,
) Attestation {
	return Attestation{
		Slot:        slot,
		Epoch:       epoch,
		Validator:   validatorAddress,
		PublicKey:   publicKey,
		BlockHash:   blockHash,
		SourceEpoch: sourceEpoch,
		TargetEpoch: targetEpoch,
		Timestamp:   now.UTC(),
	}
}

func (a Attestation) SigningPayload() attestationSigningPayload {
	return attestationSigningPayload{
		Slot:        a.Slot,
		Epoch:       a.Epoch,
		Validator:   a.Validator,
		PublicKey:   a.PublicKey,
		BlockHash:   a.BlockHash,
		SourceEpoch: a.SourceEpoch,
		TargetEpoch: a.TargetEpoch,
		Timestamp:   a.Timestamp.UTC().Unix(),
	}
}

func (a Attestation) DigestHex() (string, error) {
	return chaincrypto.HashJSON(a.SigningPayload())
}

func (a *Attestation) Sign(key *validator.LoadedKey) error {
	if a == nil {
		return fmt.Errorf("attestation is nil")
	}
	if key == nil || key.PrivateKey == nil {
		return fmt.Errorf("validator key is nil")
	}
	if a.Validator == "" {
		return fmt.Errorf("attestation validator is empty")
	}
	if a.PublicKey == "" {
		return fmt.Errorf("attestation public key is empty")
	}
	if a.BlockHash == "" {
		return fmt.Errorf("attestation block hash is empty")
	}
	if key.File.Address != a.Validator {
		return fmt.Errorf("attestation validator does not match validator key address")
	}
	if key.File.PublicKey != a.PublicKey {
		return fmt.Errorf("attestation public key does not match validator key")
	}

	digestHex, err := a.DigestHex()
	if err != nil {
		return err
	}

	sigHex, err := chaincrypto.SignHashHex(key.PrivateKey, digestHex)
	if err != nil {
		return err
	}

	a.Signature = sigHex
	return nil
}

func (a Attestation) Verify() error {
	if a.Validator == "" {
		return fmt.Errorf("attestation validator is empty")
	}
	if a.PublicKey == "" {
		return fmt.Errorf("attestation public key is empty")
	}
	if a.BlockHash == "" {
		return fmt.Errorf("attestation block hash is empty")
	}
	if a.Signature == "" {
		return fmt.Errorf("attestation signature is empty")
	}

	pub, err := chaincrypto.HexToPublicKey(a.PublicKey)
	if err != nil {
		return err
	}
	if chaincrypto.PublicKeyToAddress(pub) != a.Validator {
		return fmt.Errorf("attestation validator/public key mismatch")
	}

	digestHex, err := a.DigestHex()
	if err != nil {
		return err
	}

	ok, err := chaincrypto.VerifyHashHex(pub, digestHex, a.Signature)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("invalid attestation signature")
	}

	return nil
}
