package slashing

import "errors"

var (
	ErrSlashableBlock       = errors.New("slashable block proposal")
	ErrSlashableAttestation = errors.New("slashable attestation")
	ErrInvalidInput         = errors.New("invalid slashing input")
)
