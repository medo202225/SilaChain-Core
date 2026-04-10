package engine

import "testing"

func TestPayloadID_VersionAndIs(t *testing.T) {
	id := PayloadID{byte(PayloadV3), 1, 2, 3, 4, 5, 6, 7}

	if id.Version() != PayloadV3 {
		t.Fatalf("unexpected version: got=%d want=%d", id.Version(), PayloadV3)
	}
	if !id.Is(PayloadV2, PayloadV3) {
		t.Fatalf("expected id to match payload versions")
	}
	if id.Is(PayloadV1) {
		t.Fatalf("did not expect id to match payload v1")
	}
}

func TestPayloadID_MarshalUnmarshalText(t *testing.T) {
	id := PayloadID{1, 2, 3, 4, 5, 6, 7, 8}

	text, err := id.MarshalText()
	if err != nil {
		t.Fatalf("marshal text: %v", err)
	}

	var decoded PayloadID
	if err := decoded.UnmarshalText(text); err != nil {
		t.Fatalf("unmarshal text: %v", err)
	}
	if decoded != id {
		t.Fatalf("decoded payload id mismatch: got=%v want=%v", decoded, id)
	}
}

func TestDecodeTransactionsCopiesData(t *testing.T) {
	in := [][]byte{[]byte{1, 2, 3}, []byte{4, 5}}
	out, err := DecodeTransactions(in)
	if err != nil {
		t.Fatalf("decode transactions: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("unexpected tx count: %d", len(out))
	}
	in[0][0] = 9
	if out[0][0] != 1 {
		t.Fatalf("expected copied transaction bytes")
	}
}

func TestExecutableDataToBlockRejectsBadLogsBloomLength(t *testing.T) {
	_, err := ExecutableDataToBlockNoHash(ExecutableData{
		BlockHash: "0xblock",
		LogsBloom: []byte{1, 2, 3},
	})
	if err == nil {
		t.Fatal("expected logs bloom length error")
	}
}

func TestExecutableDataToBlock(t *testing.T) {
	data := ExecutableData{
		ParentHash:    "0xparent",
		FeeRecipient:  "SILA_fee_recipient",
		StateRoot:     "0xstate",
		ReceiptsRoot:  "0xreceipts",
		Random:        "0xrand",
		Number:        7,
		GasLimit:      30000000,
		GasUsed:       21000,
		Timestamp:     123,
		BaseFeePerGas: 10,
		BlockHash:     "0xblock",
		Transactions:  [][]byte{[]byte("tx1"), []byte("tx2")},
		LogsBloom:     make([]byte, 256),
	}

	block, err := ExecutableDataToBlock(data)
	if err != nil {
		t.Fatalf("executable data to block: %v", err)
	}
	if block.BlockHash != "0xblock" {
		t.Fatalf("unexpected block hash: got=%s want=0xblock", block.BlockHash)
	}
	if len(block.Transactions) != 2 {
		t.Fatalf("unexpected tx count: got=%d want=2", len(block.Transactions))
	}
	data.Transactions[0][0] = 'X'
	if string(block.Transactions[0]) != "tx1" {
		t.Fatalf("expected copied transaction bytes")
	}
}

func TestBlockToExecutableDataClonesBundleAndRequests(t *testing.T) {
	block := &ExecutableData{
		BlockHash:    "0xblock9",
		ParentHash:   "0xparent9",
		Transactions: [][]byte{[]byte("tx1")},
		LogsBloom:    make([]byte, 256),
	}
	bundle := &BlobsBundle{
		Commitments: [][]byte{[]byte("c1")},
		Proofs:      [][]byte{[]byte("p1")},
		Blobs:       [][]byte{[]byte("b1")},
	}
	requests := [][]byte{[]byte{1, 2}}

	env := BlockToExecutableData(block, 55, bundle, requests)
	if env == nil || env.ExecutionPayload == nil {
		t.Fatalf("expected envelope with payload")
	}
	if env.BlockValue != 55 {
		t.Fatalf("unexpected block value: got=%d want=55", env.BlockValue)
	}
	if env.ExecutionPayload.BlockHash != "0xblock9" {
		t.Fatalf("unexpected block hash: got=%s want=0xblock9", env.ExecutionPayload.BlockHash)
	}

	block.Transactions[0][0] = 'X'
	bundle.Blobs[0][0] = 'X'
	requests[0][0] = 9

	if string(env.ExecutionPayload.Transactions[0]) != "tx1" {
		t.Fatalf("expected cloned transactions")
	}
	if string(env.BlobsBundle.Blobs[0]) != "b1" {
		t.Fatalf("expected cloned blobs bundle")
	}
	if env.Requests[0][0] != 1 {
		t.Fatalf("expected cloned requests")
	}
}

func TestClientVersionString(t *testing.T) {
	v := &ClientVersionV1{
		Code:    ClientCode,
		Name:    ClientName,
		Version: "1.0.0",
		Commit:  "abc123",
	}
	if v.String() != "SI-sila-1.0.0-abc123" {
		t.Fatalf("unexpected client version string: %s", v.String())
	}
}
