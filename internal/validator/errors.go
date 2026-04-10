package validator

import "errors"

var (
	ErrNilKeyFile         = errors.New("validator key file is nil")
	ErrEmptyKeyPath       = errors.New("validator key path is empty")
	ErrInvalidKeyFile     = errors.New("invalid validator key file")
	ErrEmptyPrivateKey    = errors.New("validator private key is empty")
	ErrKeyAddressMismatch = errors.New("validator key address mismatch")
	ErrKeyFileNotFound    = errors.New("validator key file not found")
)
