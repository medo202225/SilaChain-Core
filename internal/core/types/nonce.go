package types

func ValidateNonce(expected, actual Nonce) error {
	if expected != actual {
		return ErrInvalidNonce
	}
	return nil
}
