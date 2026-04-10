package types

import "errors"

var (
	ErrNilTransaction             = errors.New("transaction is nil")
	ErrInvalidFrom                = errors.New("invalid from address")
	ErrInvalidTo                  = errors.New("invalid to address")
	ErrInvalidValue               = errors.New("invalid value")
	ErrInvalidFee                 = errors.New("invalid fee")
	ErrInvalidGasPrice            = errors.New("invalid gas price")
	ErrInvalidGasLimit            = errors.New("invalid gas limit")
	ErrInvalidNonce               = errors.New("invalid nonce")
	ErrInvalidChainID             = errors.New("invalid chain id")
	ErrInvalidTimestamp           = errors.New("invalid timestamp")
	ErrMissingPublicKey           = errors.New("missing public key")
	ErrMissingSignature           = errors.New("missing signature")
	ErrInvalidHash                = errors.New("invalid transaction hash")
	ErrPublicKeyMismatch          = errors.New("public key does not match sender address")
	ErrSignatureVerification      = errors.New("signature verification failed")
	ErrMalformedPublicKey         = errors.New("malformed public key")
	ErrMalformedSignature         = errors.New("malformed signature")
	ErrUnsupportedTransactionType = errors.New("unsupported transaction type")
)
