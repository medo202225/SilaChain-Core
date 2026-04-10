package app

import (
	"testing"
	"time"
)

func TestUniquePeers_RemovesDuplicatesAndNormalizes(t *testing.T) {
	in := []string{
		" http://127.0.0.1:8080/ ",
		"http://127.0.0.1:8080",
		"http://127.0.0.1:8081/",
		"",
		"   ",
	}

	out := UniquePeers(in, "")

	if len(out) != 2 {
		t.Fatalf("expected 2 unique peers, got %d: %#v", len(out), out)
	}
	if out[0] != "http://127.0.0.1:8080" {
		t.Fatalf("unexpected first peer: %q", out[0])
	}
	if out[1] != "http://127.0.0.1:8081" {
		t.Fatalf("unexpected second peer: %q", out[1])
	}
}

func TestPeerPolicy_BansPeerAfterThreshold(t *testing.T) {
	p := NewPeerPolicy()
	p.SetBanPolicy(3, 2*time.Minute)

	now := time.Now()
	peer := "http://127.0.0.1:8080/"

	p.ReportFailure(peer, now)
	p.ReportFailure(peer, now)
	if p.IsBanned(peer, now) {
		t.Fatalf("peer should not be banned before threshold")
	}

	p.ReportFailure(peer, now)
	if !p.IsBanned(peer, now) {
		t.Fatalf("peer should be banned at threshold")
	}
}

func TestPeerPolicy_BanExpiresAfterDuration(t *testing.T) {
	p := NewPeerPolicy()
	p.SetBanPolicy(2, 10*time.Second)

	now := time.Now()
	peer := "http://127.0.0.1:8080"

	p.ReportFailure(peer, now)
	p.ReportFailure(peer, now)

	if !p.IsBanned(peer, now) {
		t.Fatalf("peer should be banned immediately after threshold")
	}
	if p.IsBanned(peer, now.Add(11*time.Second)) {
		t.Fatalf("peer ban should expire after duration")
	}
}

func TestPeerPolicy_ReportSuccessResetsFailureState(t *testing.T) {
	p := NewPeerPolicy()
	p.SetBanPolicy(2, time.Minute)

	now := time.Now()
	peer := "http://127.0.0.1:8080"

	p.ReportFailure(peer, now)
	if got := p.FailureCount(peer); got != 1 {
		t.Fatalf("expected failure count 1, got %d", got)
	}

	p.ReportSuccess(peer)

	if got := p.FailureCount(peer); got != 0 {
		t.Fatalf("expected failure count reset to 0, got %d", got)
	}
	if p.IsBanned(peer, now) {
		t.Fatalf("peer should not remain banned after success reset")
	}
}

func TestPeerPolicy_ActivePeersFiltersBannedPeers(t *testing.T) {
	p := NewPeerPolicy()
	p.SetBanPolicy(1, time.Minute)

	now := time.Now()
	badPeer := "http://127.0.0.1:8080"
	goodPeer := "http://127.0.0.1:8081"

	p.ReportFailure(badPeer, now)

	out := p.ActivePeers([]string{badPeer, goodPeer, goodPeer + "/"}, now, "")
	if len(out) != 1 {
		t.Fatalf("expected 1 active peer, got %d: %#v", len(out), out)
	}
	if out[0] != goodPeer {
		t.Fatalf("expected good peer only, got %#v", out)
	}
}
