package slashing

import "context"

type Store interface {
	Init(ctx context.Context) error

	CheckAndRecordBlock(ctx context.Context, pubKey []byte, slot uint64, signingRoot []byte) error
	CheckAndRecordAttestation(ctx context.Context, pubKey []byte, sourceEpoch uint64, targetEpoch uint64, signingRoot []byte) error

	Close() error
}
