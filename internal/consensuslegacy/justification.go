package consensuslegacy

type JustifiedVote struct {
	Slot      Slot   `json:"slot"`
	Epoch     Epoch  `json:"epoch"`
	BlockHash string `json:"block_hash"`
	VoteCount int    `json:"vote_count"`
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
	slot Slot,
	epoch Epoch,
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

func (t *JustificationTracker) Count() int {
	if t == nil {
		return 0
	}
	return len(t.justified)
}

func (t *JustificationTracker) Last() *JustifiedVote {
	if t == nil || len(t.justified) == 0 {
		return nil
	}

	last := t.justified[len(t.justified)-1]
	return &last
}
