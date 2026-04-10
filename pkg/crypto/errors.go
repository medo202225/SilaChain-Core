package crypto

import "errors"

var (
	ErrInvalidPrivateKey = errors.New("invalid private key")
	ErrInvalidPublicKey  = errors.New("invalid public key")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrInvalidAddress    = errors.New("invalid address")
	ErrAddressMismatch   = errors.New("address does not match public key")
	ErrInvalidHex        = errors.New("invalid hex")
	ErrInvalidHashLength = errors.New("invalid hash length")
)
