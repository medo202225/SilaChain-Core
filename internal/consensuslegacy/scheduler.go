package consensuslegacy

import (
	"context"
	"log"
	"time"

	"silachain/internal/chain"
	"silachain/internal/validator"
	pkgtypes "silachain/pkg/types"
)

type Scheduler struct {
	blockchain   *chain.Blockchain
	blockTime    time.Duration
	proposerFn   func() (pkgtypes.Address, bool)
	validatorSet *validator.Set
}

func NewScheduler(
	blockchain *chain.Blockchain,
	blockTime time.Duration,
	proposerFn func() (pkgtypes.Address, bool),
	validatorSet *validator.Set,
) *Scheduler {
	return &Scheduler{
		blockchain:   blockchain,
		blockTime:    blockTime,
		proposerFn:   proposerFn,
		validatorSet: validatorSet,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	if s == nil {
		return
	}

	ticker := time.NewTicker(s.blockTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("consensus scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	if s == nil || s.blockchain == nil {
		return
	}
	if ctx.Err() != nil {
		return
	}

	proposer, ok := s.proposerAddress()
	if !ok || proposer == "" {
		log.Println("consensus scheduler: no proposer available")
		return
	}

	if _, err := s.blockchain.MinePending(proposer); err != nil {
		log.Printf("consensus scheduler: mine pending failed: %v", err)
	}
}

func (s *Scheduler) proposerAddress() (pkgtypes.Address, bool) {
	if s == nil {
		return "", false
	}

	if s.proposerFn != nil {
		if proposer, ok := s.proposerFn(); ok && proposer != "" {
			return proposer, true
		}
	}

	if s.blockchain != nil {
		return s.blockchain.CurrentProposer()
	}

	if s.blockchain != nil {
		return s.blockchain.CurrentProposer()
	}

	return "", false
}
