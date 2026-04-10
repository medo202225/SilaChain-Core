package consensuslegacy

import "sync"

type AttestationPool struct {
	mu           sync.RWMutex
	attestations []Attestation
}

func NewAttestationPool() *AttestationPool {
	return &AttestationPool{
		attestations: make([]Attestation, 0),
	}
}

func (p *AttestationPool) Add(a Attestation) error {
	if err := a.Verify(); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.attestations = append(p.attestations, a)
	return nil
}

func (p *AttestationPool) All() []Attestation {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]Attestation, len(p.attestations))
	copy(out, p.attestations)
	return out
}

func (p *AttestationPool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.attestations)
}

func (p *AttestationPool) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.attestations = make([]Attestation, 0)
}
