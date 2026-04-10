package chain

import "silachain/internal/core/state"

type Option func(*blockchainOptions)

type blockchainOptions struct {
	stateCommitment state.StateCommitment
}

func defaultBlockchainOptions() blockchainOptions {
	return blockchainOptions{
		stateCommitment: state.NewHashStateCommitment(),
	}
}

func WithStateCommitment(commitment state.StateCommitment) Option {
	return func(opts *blockchainOptions) {
		if opts == nil || commitment == nil {
			return
		}
		opts.stateCommitment = commitment
	}
}
