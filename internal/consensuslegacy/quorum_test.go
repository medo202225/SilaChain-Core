package consensuslegacy

import "testing"

func TestCheckQuorum_Reached(t *testing.T) {
	r := CheckQuorum(2, 2)
	if !r.HasQuorum {
		t.Fatalf("expected quorum to be reached")
	}
}

func TestCheckQuorum_NotReached(t *testing.T) {
	r := CheckQuorum(1, 2)
	if r.HasQuorum {
		t.Fatalf("expected quorum to not be reached")
	}
}

func TestCheckQuorum_ZeroValidators(t *testing.T) {
	r := CheckQuorum(1, 0)
	if r.HasQuorum {
		t.Fatalf("expected quorum to be false when total validators is zero")
	}
}
