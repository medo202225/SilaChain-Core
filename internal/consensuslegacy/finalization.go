package consensuslegacy

type FinalizedVote struct {
	Justified Checkpoint `json:"justified"`
	Finalized Checkpoint `json:"finalized"`
}

type FinalizationTracker struct {
	lastJustified *Checkpoint
	finalized     []FinalizedVote
}

func NewFinalizationTracker() *FinalizationTracker {
	return &FinalizationTracker{
		finalized: make([]FinalizedVote, 0),
	}
}

func (t *FinalizationTracker) AddJustified(epoch Epoch, blockHash string) bool {
	if t == nil {
		return false
	}

	current := NewCheckpoint(epoch, blockHash)

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

func (t *FinalizationTracker) Count() int {
	if t == nil {
		return 0
	}
	return len(t.finalized)
}

func (t *FinalizationTracker) LastJustified() *Checkpoint {
	if t == nil || t.lastJustified == nil {
		return nil
	}

	cp := *t.lastJustified
	return &cp
}
