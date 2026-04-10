package consensuslegacy

import (
	"fmt"
	"time"

	"silachain/internal/validator"
	chaincrypto "silachain/pkg/crypto"
	"silachain/pkg/types"
)

type Proposal struct {
	Slot      Slot          `json:"slot"`
	Epoch     Epoch         `json:"epoch"`
	Proposer  types.Address `json:"proposer"`
	PublicKey string        `json:"public_key"`
	Timestamp time.Time     `json:"timestamp"`
	Signature string        `json:"signature"`
}

type proposalSigningPayload struct {
	Slot      Slot          `json:"slot"`
	Epoch     Epoch         `json:"epoch"`
	Proposer  types.Address `json:"proposer"`
	PublicKey string        `json:"public_key"`
	Timestamp int64         `json:"timestamp_unix"`
}

func NewProposal(slot Slot, epoch Epoch, proposer types.Address, publicKey string, now time.Time) Proposal {
	return Proposal{
		Slot:      slot,
		Epoch:     epoch,
		Proposer:  proposer,
		PublicKey: publicKey,
		Timestamp: now.UTC(),
	}
}

func (p Proposal) SigningPayload() proposalSigningPayload {
	return proposalSigningPayload{
		Slot:      p.Slot,
		Epoch:     p.Epoch,
		Proposer:  p.Proposer,
		PublicKey: p.PublicKey,
		Timestamp: p.Timestamp.UTC().Unix(),
	}
}

func (p Proposal) DigestHex() (string, error) {
	payload := p.SigningPayload()
	hashHex, err := chaincrypto.HashJSON(payload)
	if err != nil {
		return "", err
	}
	return hashHex, nil
}

func (p *Proposal) Sign(key *validator.LoadedKey) error {
	if p == nil {
		return fmt.Errorf("proposal is nil")
	}
	if key == nil || key.PrivateKey == nil {
		return fmt.Errorf("validator key is nil")
	}
	if p.Proposer == "" {
		return fmt.Errorf("proposal proposer is empty")
	}
	if p.PublicKey == "" {
		return fmt.Errorf("proposal public key is empty")
	}
	if key.File.Address != p.Proposer {
		return fmt.Errorf("proposal proposer does not match validator key address")
	}
	if key.File.PublicKey != p.PublicKey {
		return fmt.Errorf("proposal public key does not match validator key")
	}

	digestHex, err := p.DigestHex()
	if err != nil {
		return err
	}

	sigHex, err := chaincrypto.SignHashHex(key.PrivateKey, digestHex)
	if err != nil {
		return err
	}

	p.Signature = sigHex
	return nil
}

func (p Proposal) Verify() error {
	if p.Proposer == "" {
		return fmt.Errorf("proposal proposer is empty")
	}
	if p.PublicKey == "" {
		return fmt.Errorf("proposal public key is empty")
	}
	if p.Signature == "" {
		return fmt.Errorf("proposal signature is empty")
	}

	pub, err := chaincrypto.HexToPublicKey(p.PublicKey)
	if err != nil {
		return err
	}
	if chaincrypto.PublicKeyToAddress(pub) != p.Proposer {
		return fmt.Errorf("proposal proposer/public key mismatch")
	}

	digestHex, err := p.DigestHex()
	if err != nil {
		return err
	}

	ok, err := chaincrypto.VerifyHashHex(pub, digestHex, p.Signature)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("invalid proposal signature")
	}

	return nil
}
