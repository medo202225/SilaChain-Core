package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type persistedPeerState struct {
	Failures    int       `json:"failures"`
	BannedUntil time.Time `json:"banned_until"`
}

type persistedPeerPolicy struct {
	Peers map[string]persistedPeerState `json:"peers"`
}

func savePeerPolicy(path string, policy *PeerPolicy) error {
	if path == "" || policy == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	snapshot := persistedPeerPolicy{
		Peers: make(map[string]persistedPeerState),
	}

	policy.mu.RLock()
	for peer, state := range policy.states {
		snapshot.Peers[peer] = persistedPeerState{
			Failures:    state.failures,
			BannedUntil: state.bannedUntil,
		}
	}
	policy.mu.RUnlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadPeerPolicy(path string, policy *PeerPolicy) error {
	if path == "" || policy == nil {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var snapshot persistedPeerPolicy
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}

	policy.mu.Lock()
	defer policy.mu.Unlock()

	if policy.states == nil {
		policy.states = make(map[string]peerState)
	}

	for peer, state := range snapshot.Peers {
		normalized := NormalizePeer(peer)
		if normalized == "" {
			continue
		}
		policy.states[normalized] = peerState{
			failures:    state.Failures,
			bannedUntil: state.BannedUntil,
		}
	}

	return nil
}
