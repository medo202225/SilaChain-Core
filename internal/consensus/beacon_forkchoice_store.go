package consensus

import "fmt"

type ForkchoiceState struct {
	HeadRoot      string `json:"head_root"`
	SafeRoot      string `json:"safe_root"`
	FinalizedRoot string `json:"finalized_root"`
}

type ForkchoiceStore struct {
	state ForkchoiceState
}

func NewForkchoiceStore() *ForkchoiceStore {
	return &ForkchoiceStore{
		state: ForkchoiceState{},
	}
}

func (s *ForkchoiceStore) UpdateFromBeaconState(beacon *BeaconStateV1) error {
	if s == nil {
		return fmt.Errorf("forkchoice store is nil")
	}
	if beacon == nil {
		return fmt.Errorf("beacon state is nil")
	}

	s.state.HeadRoot = beacon.HeadBlockRoot
	s.state.SafeRoot = beacon.SafeBlockRoot
	s.state.FinalizedRoot = beacon.FinalizedCheckpoint.Root

	return nil
}

func (s *ForkchoiceStore) State() ForkchoiceState {
	if s == nil {
		return ForkchoiceState{}
	}
	return s.state
}
