package accounts

import "errors"

var (
	ErrAccountNotFound      = errors.New("account not found")
	ErrAccountAlreadyExists = errors.New("account already exists")
	ErrInvalidAccount       = errors.New("invalid account")
	ErrInvalidNonce         = errors.New("invalid nonce")
	ErrInsufficientBalance  = errors.New("insufficient balance")
)
