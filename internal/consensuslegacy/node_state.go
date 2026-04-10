package consensuslegacy

type NodeState struct {
	AttestationPool      *AttestationPool
	JustificationTracker *JustificationTracker
	FinalizationTracker  *FinalizationTracker
}

func NewNodeState() *NodeState {
	return &NodeState{
		AttestationPool:      NewAttestationPool(),
		JustificationTracker: NewJustificationTracker(),
		FinalizationTracker:  NewFinalizationTracker(),
	}
}

func (s *NodeState) SubmitAttestation(a Attestation) error {
	if s == nil || s.AttestationPool == nil {
		return nil
	}
	return s.AttestationPool.Add(a)
}

func (s *NodeState) AllAttestations() []Attestation {
	if s == nil || s.AttestationPool == nil {
		return nil
	}
	return s.AttestationPool.All()
}

func (s *NodeState) AttestationCount() int {
	if s == nil || s.AttestationPool == nil {
		return 0
	}
	return s.AttestationPool.Count()
}

func (s *NodeState) RecordJustified(slot Slot, epoch Epoch, blockHash string, voteCount int, totalValidators int) (bool, bool) {
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

func (s *NodeState) AllJustified() []JustifiedVote {
	if s == nil || s.JustificationTracker == nil {
		return nil
	}
	return s.JustificationTracker.All()
}

func (s *NodeState) AllFinalized() []FinalizedVote {
	if s == nil || s.FinalizationTracker == nil {
		return nil
	}
	return s.FinalizationTracker.All()
}
