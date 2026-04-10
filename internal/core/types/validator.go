package types

import (
	"strings"
	"time"

	"silachain/internal/accounts"
	"silachain/internal/protocol"
	chaincrypto "silachain/pkg/crypto"
)

const (
	MaxFutureDriftSeconds = int64(15 * 60)
	MaxPastAgeSeconds     = int64(24 * 60 * 60)
	MinTransferGasLimit   = Gas(21000)
)

func isContractAddress(addr string) bool {
	addr = strings.TrimSpace(addr)
	return strings.HasPrefix(addr, "SILA_CONTRACT_")
}

func ValidateBasic(t *Transaction) error {
	if t == nil {
		return ErrNilTransaction
	}

	if err := accounts.ValidateAddress(string(t.From)); err != nil {
		return ErrInvalidFrom
	}

	switch t.NormalizedType() {
	case TypeTransfer:
		if err := accounts.ValidateAddress(string(t.To)); err != nil {
			return ErrInvalidTo
		}
		if t.Value == 0 {
			return ErrInvalidValue
		}

	case TypeContractCall:
		if !isContractAddress(string(t.To)) {
			return ErrInvalidTo
		}
		if strings.TrimSpace(t.CallMethod) == "" {
			return ErrUnsupportedTransactionType
		}

	default:
		return ErrUnsupportedTransactionType
	}

	if t.Fee == 0 && t.GasPrice == 0 {
		return ErrInvalidFee
	}

	if t.Fee > 0 {
		if err := ValidateFee(t.Fee); err != nil {
			return err
		}
	}

	if t.GasPrice > 0 && t.GasLimit == 0 {
		return ErrInvalidGasLimit
	}

	if t.GasLimit > 0 && t.GasLimit < MinTransferGasLimit {
		return ErrInvalidGasLimit
	}

	if !protocol.IsSupportedChainID(t.ChainID) {
		return ErrInvalidChainID
	}
	if t.Timestamp <= 0 {
		return ErrInvalidTimestamp
	}

	now := time.Now().Unix()
	if int64(t.Timestamp) > now+MaxFutureDriftSeconds {
		return ErrInvalidTimestamp
	}
	if int64(t.Timestamp) < now-MaxPastAgeSeconds {
		return ErrInvalidTimestamp
	}

	if strings.TrimSpace(t.PublicKey) == "" {
		return ErrMissingPublicKey
	}
	if strings.TrimSpace(t.Signature) == "" {
		return ErrMissingSignature
	}
	if strings.TrimSpace(string(t.Hash)) == "" {
		return ErrInvalidHash
	}

	pub := strings.TrimSpace(t.PublicKey)
	sig := strings.TrimSpace(t.Signature)

	pubBytes, err := chaincrypto.DecodeHex(pub)
	if err != nil {
		return ErrMalformedPublicKey
	}
	if len(pubBytes) != 65 {
		return ErrMalformedPublicKey
	}

	sigBytes, err := chaincrypto.DecodeHex(sig)
	if err != nil {
		return ErrMalformedSignature
	}
	if len(sigBytes) != 65 {
		return ErrMalformedSignature
	}

	return nil
}

func Validate(t *Transaction) error {
	if err := ValidateBasic(t); err != nil {
		return err
	}

	expectedHash, err := ComputeHash(t)
	if err != nil {
		return err
	}
	if t.Hash != "" && t.Hash != expectedHash {
		return ErrInvalidHash
	}

	return VerifySignature(t)
}
