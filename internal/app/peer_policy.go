package app

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type peerState struct {
	failures    int
	bannedUntil time.Time
}

type PeerPolicy struct {
	mu          sync.RWMutex
	states      map[string]peerState
	threshold   int
	banDuration time.Duration
	path        string
}

func NewPeerPolicy() *PeerPolicy {
	return &PeerPolicy{
		states:      map[string]peerState{},
		threshold:   3,
		banDuration: time.Minute,
	}
}

func NormalizePeer(peer string) string {
	peer = strings.TrimSpace(peer)
	if peer == "" {
		return ""
	}
	if !strings.HasPrefix(peer, "http://") && !strings.HasPrefix(peer, "https://") {
		peer = "http://" + peer
	}
	return strings.TrimRight(peer, "/")
}

func UniquePeers(peers []string, selfURL string) []string {
	self := NormalizePeer(selfURL)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(peers))
	for _, peer := range peers {
		n := NormalizePeer(peer)
		if n == "" || n == self {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func (p *PeerPolicy) SetBanPolicy(threshold int, duration time.Duration) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.threshold = threshold
	p.banDuration = duration
}

func (p *PeerPolicy) ReportFailure(peer string, now time.Time) {
	if p == nil {
		return
	}
	peer = NormalizePeer(peer)

	p.mu.Lock()
	defer p.mu.Unlock()

	state := p.states[peer]
	state.failures++
	if p.threshold > 0 && state.failures >= p.threshold {
		state.bannedUntil = now.Add(p.banDuration)
	}
	p.states[peer] = state
}

func (p *PeerPolicy) ReportSuccess(peer string) {
	if p == nil {
		return
	}
	peer = NormalizePeer(peer)

	p.mu.Lock()
	defer p.mu.Unlock()

	state := p.states[peer]
	state.failures = 0
	state.bannedUntil = time.Time{}
	p.states[peer] = state
}

func (p *PeerPolicy) FailureCount(peer string) int {
	if p == nil {
		return 0
	}
	peer = NormalizePeer(peer)

	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.states[peer].failures
}

func (p *PeerPolicy) IsBanned(peer string, now time.Time) bool {
	if p == nil {
		return false
	}
	peer = NormalizePeer(peer)

	p.mu.RLock()
	defer p.mu.RUnlock()

	state := p.states[peer]
	return !state.bannedUntil.IsZero() && now.Before(state.bannedUntil)
}

func (p *PeerPolicy) ActivePeers(peers []string, now time.Time, selfURL string) []string {
	if p == nil {
		return UniquePeers(peers, selfURL)
	}

	candidates := UniquePeers(peers, selfURL)
	out := make([]string, 0, len(candidates))

	for _, peer := range candidates {
		if !p.IsBanned(peer, now) {
			out = append(out, peer)
		}
	}

	return out
}

type PeerPolicyEntry struct {
	Peer        string    `json:"peer"`
	Failures    int       `json:"failures"`
	Banned      bool      `json:"banned"`
	BannedUntil time.Time `json:"banned_until"`
}

type PeerPolicySnapshot struct {
	Threshold   int               `json:"threshold"`
	BanDuration string            `json:"ban_duration"`
	Entries     []PeerPolicyEntry `json:"entries"`
}

func (p *PeerPolicy) Snapshot(now time.Time, peers []string, selfURL string) PeerPolicySnapshot {
	if p == nil {
		return PeerPolicySnapshot{}
	}

	candidates := UniquePeers(peers, selfURL)

	p.mu.RLock()
	defer p.mu.RUnlock()

	out := PeerPolicySnapshot{
		Threshold:   p.threshold,
		BanDuration: p.banDuration.String(),
		Entries:     make([]PeerPolicyEntry, 0, len(candidates)),
	}

	for _, peer := range candidates {
		state := p.states[peer]
		out.Entries = append(out.Entries, PeerPolicyEntry{
			Peer:        peer,
			Failures:    state.failures,
			Banned:      !state.bannedUntil.IsZero() && now.Before(state.bannedUntil),
			BannedUntil: state.bannedUntil,
		})
	}

	return out
}
