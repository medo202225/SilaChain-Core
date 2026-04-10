package engineapi

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/forkchoice"
)

func TestPayloadMetadata_SnapshotAndRestore_RetainsMetadataLookup(t *testing.T) {
	store, err := forkchoice.New(blockassembly.Head{
		Number:    10,
		Hash:      "0xhead10",
		StateRoot: "0xstate10",
		BaseFee:   1,
	})
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	svc, err := NewBuilderService(store, &stubAssembler{})
	if err != nil {
		t.Fatalf("create builder service: %v", err)
	}

	buildResult, err := svc.ForkchoiceUpdatedWithAttributes(ForkchoiceState{
		HeadBlockHash:      "0xhead10",
		SafeBlockHash:      "0xhead10",
		FinalizedBlockHash: "0xhead10",
	}, &blockassembly.PayloadAttributes{
		Timestamp:         61,
		FeeRecipient:      "SILA_fee_recipient_snapshot",
		Random:            "SILA_random_snapshot",
		SuggestedGasLimit: 30000000,
	})
	if err != nil {
		t.Fatalf("forkchoice updated with attrs: %v", err)
	}

	metaBeforeSnapshot, err := svc.GetPayloadMetadata(buildResult.PayloadID)
	if err != nil {
		t.Fatalf("get metadata before snapshot: %v", err)
	}
	if metaBeforeSnapshot.PayloadID != buildResult.PayloadID {
		t.Fatalf("unexpected metadata payload id: got=%s want=%s", metaBeforeSnapshot.PayloadID, buildResult.PayloadID)
	}
	if metaBeforeSnapshot.BlockHash == "" {
		t.Fatalf("expected metadata block hash to be populated before snapshot")
	}

	snapshot := svc.SnapshotPayloadMetadata()

	restoredStore, err := forkchoice.New(blockassembly.Head{
		Number:    10,
		Hash:      "0xhead10",
		StateRoot: "0xstate10",
		BaseFee:   1,
	})
	if err != nil {
		t.Fatalf("create restored store: %v", err)
	}

	restored, err := NewBuilderService(restoredStore, &stubAssembler{})
	if err != nil {
		t.Fatalf("create restored builder service: %v", err)
	}

	restored.RestorePayloadMetadata(snapshot)

	metaByID, err := restored.GetPayloadMetadata(buildResult.PayloadID)
	if err != nil {
		t.Fatalf("get restored metadata by id: %v", err)
	}
	if metaByID.PayloadID != metaBeforeSnapshot.PayloadID {
		t.Fatalf("unexpected restored metadata payload id: got=%s want=%s", metaByID.PayloadID, metaBeforeSnapshot.PayloadID)
	}
	if metaByID.BlockHash != metaBeforeSnapshot.BlockHash {
		t.Fatalf("unexpected restored metadata block hash: got=%s want=%s", metaByID.BlockHash, metaBeforeSnapshot.BlockHash)
	}
	if metaByID.BlockNumber != metaBeforeSnapshot.BlockNumber {
		t.Fatalf("unexpected restored metadata block number: got=%d want=%d", metaByID.BlockNumber, metaBeforeSnapshot.BlockNumber)
	}
	if metaByID.TxCount != metaBeforeSnapshot.TxCount {
		t.Fatalf("unexpected restored metadata tx count: got=%d want=%d", metaByID.TxCount, metaBeforeSnapshot.TxCount)
	}

	metaByHash, ok := restored.GetPayloadMetadataByBlockHash(metaBeforeSnapshot.BlockHash)
	if !ok {
		t.Fatalf("expected restored metadata lookup by block hash to succeed")
	}
	if metaByHash.PayloadID != buildResult.PayloadID {
		t.Fatalf("unexpected restored metadata payload id by hash: got=%s want=%s", metaByHash.PayloadID, buildResult.PayloadID)
	}
}
