package accounts

import (
	"errors"
	"strings"
)

func ValidateAddress(address string) error {
	addr := strings.TrimSpace(address)
	if addr == "" {
		return errors.New("empty address")
	}
	if !strings.HasPrefix(addr, "SILA_") {
		return errors.New("address must start with SILA_")
	}
	if len(addr) <= len("SILA_") {
		return errors.New("invalid address length")
	}
	return nil
}
