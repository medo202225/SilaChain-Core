package types

import (
	"silachain/internal/protocol"
)

func ValidateFee(fee Amount) error {
	if fee < protocol.DefaultMainnetParams().MinFee {
		return ErrInvalidFee
	}
	return nil
}
