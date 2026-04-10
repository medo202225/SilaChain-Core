package consensus

import (
	"fmt"
	"sync"
	"time"

	chaincrypto "silachain/pkg/crypto"
	pkgtypes "silachain/pkg/types"
)

type Attestation struct {
	Slot        uint64           `json:"slot"`
	Epoch       uint64           `json:"epoch"`
	Validator   pkgtypes.Address `json:"validator"`
	PublicKey   string           `json:"public_key"`
	BlockHash   string           `json:"block_hash"`
	SourceEpoch uint64           `json:"source_epoch"`
	TargetEpoch uint64           `json:"target_epoch"`
	Timestamp   time.Time        `json:"timestamp"`
	Signature   string           `json:"signature"`
}

type attestationSigningPayload struct {
	Slot        uint64           `json:"slot"`
	Epoch       uint64           `json:"epoch"`
	Validator   pkgtypes.Address `json:"validator"`
	PublicKey   string           `json:"public_key"`
	BlockHash   string           `json:"block_hash"`
	SourceEpoch uint64           `json:"source_epoch"`
	TargetEpoch uint64           `json:"target_epoch"`
	Timestamp   int64            `json:"timestamp_unix"`
}

func (a Attestation) SigningPayload() attestationSigningPayload {
	return attestationSigningPayload{
		Slot:        a.Slot,
		Epoch:       a.Epoch,
		Validator:   a.Validator,
		PublicKey:   a.PublicKey,
		BlockHash:   a.BlockHash,
		SourceEpoch: a.SourceEpoch,
		TargetEpoch: a.TargetEpoch,
		Timestamp:   a.Timestamp.UTC().Unix(),
	}
}

func (a Attestation) DigestHex() (string, error) {
	return chaincrypto.HashJSON(a.SigningPayload())
}

func (a Attestation) Verify() error {
	if a.Validator == "" {
		return fmt.Errorf("attestation validator is empty")
	}
	if a.PublicKey == "" {
		return fmt.Errorf("attestation public key is empty")
	}
	if a.BlockHash == "" {
		return fmt.Errorf("attestation block hash is empty")
	}
	if a.Signature == "" {
		return fmt.Errorf("attestation signature is empty")
	}

	pub, err := chaincrypto.HexToPublicKey(a.PublicKey)
	if err != nil {
		return err
	}
	if chaincrypto.PublicKeyToAddress(pub) != a.Validator {
		return fmt.Errorf("attestation validator/public key mismatch")
	}

	digestHex, err := a.DigestHex()
	if err != nil {
		return err
	}

	ok, err := chaincrypto.VerifyHashHex(pub, digestHex, a.Signature)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("invalid attestation signature")
	}

	return nil
}

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

type AttestationAggregate struct {
	Slot       uint64             `json:"slot"`
	Epoch      uint64             `json:"epoch"`
	BlockHash  string             `json:"block_hash"`
	Validators []pkgtypes.Address `json:"validators"`
	VoteCount  int                `json:"vote_count"`
}

func AggregateAttestations(attestations []Attestation) []AttestationAggregate {
	type key struct {
		slot      uint64
		epoch     uint64
		blockHash string
	}

	grouped := make(map[key]map[pkgtypes.Address]struct{})

	for _, a := range attestations {
		k := key{
			slot:      a.Slot,
			epoch:     a.Epoch,
			blockHash: a.BlockHash,
		}
		if grouped[k] == nil {
			grouped[k] = make(map[pkgtypes.Address]struct{})
		}
		grouped[k][a.Validator] = struct{}{}
	}

	out := make([]AttestationAggregate, 0, len(grouped))
	for k, validatorSet := range grouped {
		validators := make([]pkgtypes.Address, 0, len(validatorSet))
		for v := range validatorSet {
			validators = append(validators, v)
		}

		out = append(out, AttestationAggregate{
			Slot:       k.slot,
			Epoch:      k.epoch,
			BlockHash:  k.blockHash,
			Validators: validators,
			VoteCount:  len(validators),
		})
	}

	return out
}

type QuorumResult struct {
	VoteCount            int  `json:"vote_count"`
	TotalValidators      int  `json:"total_validators"`
	HasQuorum            bool `json:"has_quorum"`
	ThresholdNumerator   int  `json:"threshold_numerator"`
	ThresholdDenominator int  `json:"threshold_denominator"`
}

func CheckQuorum(voteCount int, totalValidators int) QuorumResult {
	result := QuorumResult{
		VoteCount:            voteCount,
		TotalValidators:      totalValidators,
		ThresholdNumerator:   2,
		ThresholdDenominator: 3,
	}

	if totalValidators <= 0 {
		result.HasQuorum = false
		return result
	}

	result.HasQuorum = voteCount*3 >= totalValidators*2
	return result
}

type VoteCheckpoint struct {
	Epoch     uint64 `json:"epoch"`
	BlockHash string `json:"block_hash"`
}

type JustifiedVote struct {
	Slot      uint64 `json:"slot"`
	Epoch     uint64 `json:"epoch"`
	BlockHash string `json:"block_hash"`
	VoteCount int    `json:"vote_count"`
}

type FinalizedVote struct {
	Justified VoteCheckpoint `json:"justified"`
	Finalized VoteCheckpoint `json:"finalized"`
}

type JustificationTracker struct {
	justified []JustifiedVote
}

func NewJustificationTracker() *JustificationTracker {
	return &JustificationTracker{
		justified: make([]JustifiedVote, 0),
	}
}

func (t *JustificationTracker) AddIfQuorum(
	slot uint64,
	epoch uint64,
	blockHash string,
	voteCount int,
	totalValidators int,
) bool {
	if t == nil {
		return false
	}

	quorum := CheckQuorum(voteCount, totalValidators)
	if !quorum.HasQuorum {
		return false
	}

	for _, existing := range t.justified {
		if existing.Epoch == epoch && existing.BlockHash == blockHash {
			return false
		}
	}

	t.justified = append(t.justified, JustifiedVote{
		Slot:      slot,
		Epoch:     epoch,
		BlockHash: blockHash,
		VoteCount: voteCount,
	})
	return true
}

func (t *JustificationTracker) All() []JustifiedVote {
	if t == nil {
		return nil
	}

	out := make([]JustifiedVote, len(t.justified))
	copy(out, t.justified)
	return out
}

type FinalizationTracker struct {
	lastJustified *VoteCheckpoint
	finalized     []FinalizedVote
}

func NewFinalizationTracker() *FinalizationTracker {
	return &FinalizationTracker{
		finalized: make([]FinalizedVote, 0),
	}
}

func (t *FinalizationTracker) AddJustified(epoch uint64, blockHash string) bool {
	if t == nil {
		return false
	}

	current := VoteCheckpoint{
		Epoch:     epoch,
		BlockHash: blockHash,
	}

	if t.lastJustified == nil {
		t.lastJustified = &current
		return false
	}

	previous := *t.lastJustified
	didFinalize := false

	if current.Epoch > previous.Epoch && current.BlockHash == previous.BlockHash {
		alreadyFinalized := false
		for _, item := range t.finalized {
			if item.Justified.Epoch == previous.Epoch &&
				item.Justified.BlockHash == previous.BlockHash &&
				item.Finalized.Epoch == current.Epoch &&
				item.Finalized.BlockHash == current.BlockHash {
				alreadyFinalized = true
				break
			}
		}

		if !alreadyFinalized {
			t.finalized = append(t.finalized, FinalizedVote{
				Justified: previous,
				Finalized: current,
			})
			didFinalize = true
		}
	}

	t.lastJustified = &current
	return didFinalize
}

func (t *FinalizationTracker) All() []FinalizedVote {
	if t == nil {
		return nil
	}

	out := make([]FinalizedVote, len(t.finalized))
	copy(out, t.finalized)
	return out
}

type ReadState struct {
	AttestationPool      *AttestationPool
	JustificationTracker *JustificationTracker
	FinalizationTracker  *FinalizationTracker
}

func NewReadState() *ReadState {
	return &ReadState{
		AttestationPool:      NewAttestationPool(),
		JustificationTracker: NewJustificationTracker(),
		FinalizationTracker:  NewFinalizationTracker(),
	}
}

func (s *ReadState) SubmitAttestation(a Attestation) error {
	if s == nil || s.AttestationPool == nil {
		return nil
	}
	return s.AttestationPool.Add(a)
}

func (s *ReadState) AllAttestations() []Attestation {
	if s == nil || s.AttestationPool == nil {
		return nil
	}
	return s.AttestationPool.All()
}

func (s *ReadState) RecordJustified(slot uint64, epoch uint64, blockHash string, voteCount int, totalValidators int) (bool, bool) {
	if s == nil || s.JustificationTracker == nil {
		return false, false
	}

	justified := s.JustificationTracker.AddIfQuorum(slot, epoch, blockHash, voteCount, totalValidators)
	if !justified {
		return false, false
	}

	if s.FinalizationTracker == nil {
		return true, false
	}

	finalized := s.FinalizationTracker.AddJustified(epoch, blockHash)
	return true, finalized
}

func (s *ReadState) AllJustified() []JustifiedVote {
	if s == nil || s.JustificationTracker == nil {
		return nil
	}
	return s.JustificationTracker.All()
}

func (s *ReadState) AllFinalized() []FinalizedVote {
	if s == nil || s.FinalizationTracker == nil {
		return nil
	}
	return s.FinalizationTracker.All()
}
