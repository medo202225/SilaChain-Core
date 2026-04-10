package catalyst

import (
	"context"
	"testing"

	beaconengine "silachain/beacon/engine"
)

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

func TestForkchoiceUpdatedV1RejectsWithdrawals(t *testing.T) {
	api := newConsensusAPIWithoutHeartbeat(&testBackend{})
	withdrawals := []string{"w1"}

	_, err := api.ForkchoiceUpdatedV1(context.Background(), beaconengine.ForkchoiceStateV1{
		HeadBlockHash: "0xhead",
	}, &beaconengine.PayloadAttributes{
		Timestamp:   100,
		Withdrawals: withdrawals,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestForkchoiceUpdatedV2RequiresWithdrawalsAtShanghai(t *testing.T) {
	api := newConsensusAPIWithoutHeartbeat(&testBackend{})

	_, err := api.ForkchoiceUpdatedV2(context.Background(), beaconengine.ForkchoiceStateV1{
		HeadBlockHash: "0xhead",
	}, &beaconengine.PayloadAttributes{
		Timestamp: 300,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestForkchoiceUpdatedV3RequiresBeaconRoot(t *testing.T) {
	api := newConsensusAPIWithoutHeartbeat(&testBackend{})
	withdrawals := []string{"w1"}

	_, err := api.ForkchoiceUpdatedV3(context.Background(), beaconengine.ForkchoiceStateV1{
		HeadBlockHash: "0xhead",
	}, &beaconengine.PayloadAttributes{
		Timestamp:   400,
		Withdrawals: withdrawals,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestForkchoiceUpdatedV4RequiresSlotNumber(t *testing.T) {
	api := newConsensusAPIWithoutHeartbeat(&testBackend{})
	withdrawals := []string{"w1"}
	root := "0xbeacon"

	_, err := api.ForkchoiceUpdatedV4(context.Background(), beaconengine.ForkchoiceStateV1{
		HeadBlockHash: "0xhead",
	}, &beaconengine.PayloadAttributes{
		Timestamp:   500,
		Withdrawals: withdrawals,
		BeaconRoot:  &root,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestForkchoiceUpdatedReturnsValid(t *testing.T) {
	api := newConsensusAPIWithoutHeartbeat(&testBackend{})

	res, err := api.ForkchoiceUpdatedV1(context.Background(), beaconengine.ForkchoiceStateV1{
		HeadBlockHash: "0xhead",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PayloadStatus.Status != beaconengine.VALID {
		t.Fatalf("unexpected status: got=%s want=%s", res.PayloadStatus.Status, beaconengine.VALID)
	}
	if res.PayloadStatus.LatestValidHash == nil || *res.PayloadStatus.LatestValidHash != "0xhead" {
		t.Fatalf("unexpected latest valid hash")
	}
}

func TestNewPayloadV1RejectsWithdrawals(t *testing.T) {
	api := newConsensusAPIWithoutHeartbeat(&testBackend{})

	_, err := api.NewPayloadV1(context.Background(), beaconengine.ExecutableData{
		BlockHash:   "0xblock",
		Withdrawals: []string{"w1"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewPayloadV3RequiresBeaconRoot(t *testing.T) {
	api := newConsensusAPIWithoutHeartbeat(&testBackend{})
	blobGas := uint64(1)
	excessBlobGas := uint64(2)

	_, err := api.NewPayloadV3(context.Background(), beaconengine.ExecutableData{
		BlockHash:     "0xblock",
		Timestamp:     400,
		Withdrawals:   []string{"w1"},
		BlobGasUsed:   &blobGas,
		ExcessBlobGas: &excessBlobGas,
	}, []string{"0xvh1"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewPayloadV5RequiresSlotNumber(t *testing.T) {
	api := newConsensusAPIWithoutHeartbeat(&testBackend{})
	blobGas := uint64(1)
	excessBlobGas := uint64(2)
	root := "0xbeacon"

	_, err := api.NewPayloadV5(context.Background(), beaconengine.ExecutableData{
		BlockHash:     "0xblock",
		Timestamp:     500,
		Withdrawals:   []string{"w1"},
		BlobGasUsed:   &blobGas,
		ExcessBlobGas: &excessBlobGas,
	}, []string{"0xvh1"}, &root, [][]byte{{1, 2}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewPayloadReturnsValid(t *testing.T) {
	api := newConsensusAPIWithoutHeartbeat(&testBackend{})

	res, err := api.NewPayloadV1(context.Background(), beaconengine.ExecutableData{
		BlockHash: "0xblock",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != beaconengine.VALID {
		t.Fatalf("unexpected status: got=%s want=%s", res.Status, beaconengine.VALID)
	}
	if res.LatestValidHash == nil || *res.LatestValidHash != "0xblock" {
		t.Fatalf("unexpected latest valid hash")
	}
}
