package consensus

import (
	"context"
	"crypto/sha256"
	"encoding/binary"

	validatorclient "silachain/internal/validatorclient"
)

type StaticDutyProvider struct {
	state         *BeaconStateV1
	slotsPerEpoch uint64
}

func NewStaticDutyProvider(state *BeaconStateV1, slotsPerEpoch uint64) *StaticDutyProvider {
	if slotsPerEpoch == 0 {
		slotsPerEpoch = 32
	}
	return &StaticDutyProvider{
		state:         state,
		slotsPerEpoch: slotsPerEpoch,
	}
}

func (p *StaticDutyProvider) AdvanceToSlot(slot uint64) {
	if p == nil || p.state == nil {
		return
	}
	p.state.AdvanceSlot(slot, p.slotsPerEpoch)
}

func proposalSigningRoot(slot uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, slot)
	sum := sha256.Sum256(append([]byte("proposal:"), buf...))
	return sum[:]
}

func attestationSigningRoot(sourceEpoch uint64, targetEpoch uint64) []byte {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[:8], sourceEpoch)
	binary.BigEndian.PutUint64(buf[8:], targetEpoch)
	sum := sha256.Sum256(append([]byte("attestation:"), buf...))
	return sum[:]
}

func (p *StaticDutyProvider) NextProposalDuty(ctx context.Context) (*validatorclient.ProposalDuty, bool, error) {
	_ = ctx
	if p == nil || p.state == nil {
		return nil, false, nil
	}

	proposer := p.state.ProposerForCurrentSlot()
	if proposer == nil {
		return nil, false, nil
	}

	slot := p.state.Slot
	return &validatorclient.ProposalDuty{
		PublicKey:   proposer.PublicKey,
		Slot:        slot,
		SigningRoot: proposalSigningRoot(slot),
	}, true, nil
}

func (p *StaticDutyProvider) NextAttestationDuty(ctx context.Context) (*validatorclient.AttestationDuty, bool, error) {
	_ = ctx
	if p == nil || p.state == nil {
		return nil, false, nil
	}

	attesters := p.state.AttestersForCurrentSlot()
	if len(attesters) == 0 {
		return nil, false, nil
	}

	target := p.state.Epoch
	source := uint64(0)
	if target > 0 {
		source = target - 1
	}

	// ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬ط¢آ ط·آ·ط¢آ·ط·آ¢ط¢آ¹ط·آ·ط¢آ¸ط·آ¸ط¢آ¹ط·آ·ط¢آ·ط·آ¢ط¢آ¯ duty ط·آ·ط¢آ·ط·آ¢ط¢آ¹ط·آ·ط¢آ·ط·آ¢ط¢آ§ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬ط¢آ¦ط·آ·ط¢آ·ط·آ¢ط¢آ© ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬أ¢â‚¬ع†ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬أ¢â‚¬ع†ط·آ·ط¢آ¸ط£آ¢أ¢â‚¬ع‘ط¢آ¬ slot/epoch ط·آ·ط¢آ·ط·آ¢ط¢آ§ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬أ¢â‚¬ع†ط·آ·ط¢آ·ط·آ¢ط¢آ­ط·آ·ط¢آ·ط·آ¢ط¢آ§ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬أ¢â‚¬ع†ط·آ·ط¢آ¸ط·آ¸ط¢آ¹ط·آ·ط¢آ¸ط·آ¸ط¢آ¹ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬ط¢آ ط·آ·ط¢آ·ط·آ¥أ¢â‚¬â„¢ ط·آ·ط¢آ¸ط·آ«أ¢â‚¬آ ط·آ·ط¢آ·ط·آ¢ط¢آ§ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬أ¢â‚¬ع†ط·آ·ط¢آ¸ط£آ¢أ¢â‚¬ع‘ط¢آ¬ scheduler ط·آ·ط¢آ·ط·آ¢ط¢آ³ط·آ·ط¢آ¸ط·آ¸ط¢آ¹ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬ط¹â€کط·آ·ط¢آ·ط·آ¢ط¢آ±ط·آ·ط¢آ·ط·آ¢ط¢آ± ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬ط·إ’ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬أ¢â‚¬ع† local validator ط·آ·ط¢آ·ط·آ¢ط¢آ¹ط·آ·ط¢آ·ط·آ¢ط¢آ¶ط·آ·ط¢آ¸ط·آ«أ¢â‚¬آ  ط·آ·ط¢آ¸ط·آ¸ط¢آ¾ط·آ·ط¢آ¸ط·آ¸ط¢آ¹ committee ط·آ·ط¢آ·ط·آ¢ط¢آ£ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬ط¢آ¦ ط·آ·ط¢آ¸ط£آ¢أ¢â€ڑآ¬أ¢â‚¬ع†ط·آ·ط¢آ·ط·آ¢ط¢آ§.
	return &validatorclient.AttestationDuty{
		PublicKey:   "",
		SourceEpoch: source,
		TargetEpoch: target,
		SigningRoot: attestationSigningRoot(source, target),
	}, true, nil
}
