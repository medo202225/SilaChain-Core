package catalyst

import "testing"

type testBackend struct {
	callback func(invalidHash string, originHash string)
}

func (b *testBackend) SetBadBlockCallback(cb func(invalidHash string, originHash string)) {
	b.callback = cb
}

type testNode struct {
	apis []API
}

func (n *testNode) RegisterAPIs(apis []API) {
	n.apis = append(n.apis, apis...)
}

func TestRegister(t *testing.T) {
	node := &testNode{}
	backend := &testBackend{}

	if err := Register(node, backend); err != nil {
		t.Fatalf("register: %v", err)
	}
	if len(node.apis) != 1 {
		t.Fatalf("unexpected api count: got=%d want=1", len(node.apis))
	}
	if node.apis[0].Namespace != "engine" {
		t.Fatalf("unexpected namespace: got=%s want=engine", node.apis[0].Namespace)
	}
	if !node.apis[0].Authenticated {
		t.Fatalf("expected authenticated api")
	}
}

func TestNewConsensusAPIWithoutHeartbeat(t *testing.T) {
	backend := &testBackend{}

	api := newConsensusAPIWithoutHeartbeat(backend)
	if api == nil {
		t.Fatal("expected api")
	}
	if api.remoteBlocks == nil {
		t.Fatal("expected remoteBlocks queue")
	}
	if api.localBlocks == nil {
		t.Fatal("expected localBlocks queue")
	}
	if api.invalidBlocksHits == nil {
		t.Fatal("expected invalidBlocksHits map")
	}
	if api.invalidTipsets == nil {
		t.Fatal("expected invalidTipsets map")
	}
	if backend.callback == nil {
		t.Fatal("expected bad block callback to be registered")
	}
}

func TestSetInvalidAncestor(t *testing.T) {
	api := newConsensusAPIWithoutHeartbeat(&testBackend{})

	api.setInvalidAncestor("0xbad", "0xorigin")

	if api.invalidTipsets["0xorigin"] != "0xbad" {
		t.Fatalf("unexpected invalid tipset mapping")
	}
	if api.invalidBlocksHits["0xbad"] != 1 {
		t.Fatalf("unexpected invalid block hit count: got=%d want=1", api.invalidBlocksHits["0xbad"])
	}
}
