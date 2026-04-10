package engine

import (
	"encoding/hex"
	"fmt"
	"slices"
)

type PayloadVersion byte

var (
	PayloadV1 PayloadVersion = 0x1
	PayloadV2 PayloadVersion = 0x2
	PayloadV3 PayloadVersion = 0x3
	PayloadV4 PayloadVersion = 0x4
)

type PayloadAttributes struct {
	Timestamp             uint64   `json:"timestamp"`
	Random                string   `json:"prevRandao"`
	SuggestedFeeRecipient string   `json:"suggestedFeeRecipient"`
	Withdrawals           []string `json:"withdrawals,omitempty"`
	BeaconRoot            *string  `json:"parentBeaconBlockRoot,omitempty"`
	SlotNumber            *uint64  `json:"slotNumber,omitempty"`
}

type ExecutableData struct {
	ParentHash    string   `json:"parentHash"`
	FeeRecipient  string   `json:"feeRecipient"`
	StateRoot     string   `json:"stateRoot"`
	ReceiptsRoot  string   `json:"receiptsRoot"`
	LogsBloom     []byte   `json:"logsBloom"`
	Random        string   `json:"prevRandao"`
	Number        uint64   `json:"blockNumber"`
	GasLimit      uint64   `json:"gasLimit"`
	GasUsed       uint64   `json:"gasUsed"`
	Timestamp     uint64   `json:"timestamp"`
	ExtraData     []byte   `json:"extraData"`
	BaseFeePerGas uint64   `json:"baseFeePerGas"`
	BlockHash     string   `json:"blockHash"`
	Transactions  [][]byte `json:"transactions"`
	Withdrawals   []string `json:"withdrawals,omitempty"`
	BlobGasUsed   *uint64  `json:"blobGasUsed,omitempty"`
	ExcessBlobGas *uint64  `json:"excessBlobGas,omitempty"`
	SlotNumber    *uint64  `json:"slotNumber,omitempty"`
}

type executableDataMarshaling struct {
	Number        uint64
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64
	BaseFeePerGas uint64
	ExtraData     []byte
	LogsBloom     []byte
	Transactions  [][]byte
	BlobGasUsed   *uint64
	ExcessBlobGas *uint64
	SlotNumber    *uint64
}

type StatelessPayloadStatusV1 struct {
	Status          string  `json:"status"`
	StateRoot       string  `json:"stateRoot"`
	ReceiptsRoot    string  `json:"receiptsRoot"`
	ValidationError *string `json:"validationError"`
}

type ExecutionPayloadEnvelope struct {
	ExecutionPayload *ExecutableData `json:"executionPayload"`
	BlockValue       uint64          `json:"blockValue"`
	BlobsBundle      *BlobsBundle    `json:"blobsBundle,omitempty"`
	Requests         [][]byte        `json:"executionRequests,omitempty"`
	Override         bool            `json:"shouldOverrideBuilder"`
	Witness          *[]byte         `json:"witness,omitempty"`
}

type executionPayloadEnvelopeMarshaling struct {
	BlockValue uint64
	Requests   [][]byte
}

type BlobsBundle struct {
	Commitments [][]byte `json:"commitments"`
	Proofs      [][]byte `json:"proofs"`
	Blobs       [][]byte `json:"blobs"`
}

type BlobAndProofV1 struct {
	Blob  []byte `json:"blob"`
	Proof []byte `json:"proof"`
}

type BlobAndProofV2 struct {
	Blob       []byte   `json:"blob"`
	CellProofs [][]byte `json:"proofs"`
}

type PayloadStatusV1 struct {
	Status          string  `json:"status"`
	Witness         *[]byte `json:"witness,omitempty"`
	LatestValidHash *string `json:"latestValidHash"`
	ValidationError *string `json:"validationError"`
}

type TransitionConfigurationV1 struct {
	TerminalTotalDifficulty uint64 `json:"terminalTotalDifficulty"`
	TerminalBlockHash       string `json:"terminalBlockHash"`
	TerminalBlockNumber     uint64 `json:"terminalBlockNumber"`
}

type PayloadID [8]byte

func (b PayloadID) Version() PayloadVersion {
	return PayloadVersion(b[0])
}

func (b PayloadID) Is(versions ...PayloadVersion) bool {
	return slices.Contains(versions, b.Version())
}

func (b PayloadID) String() string {
	return "0x" + hex.EncodeToString(b[:])
}

func (b PayloadID) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b *PayloadID) UnmarshalText(input []byte) error {
	raw := input
	if len(raw) >= 2 && string(raw[:2]) == "0x" {
		raw = raw[2:]
	}
	decoded, err := hex.DecodeString(string(raw))
	if err != nil {
		return fmt.Errorf("invalid payload id %q: %w", string(input), err)
	}
	if len(decoded) != len(b[:]) {
		return fmt.Errorf("invalid payload id length: %d", len(decoded))
	}
	copy(b[:], decoded)
	return nil
}

type ForkChoiceResponse struct {
	PayloadStatus PayloadStatusV1 `json:"payloadStatus"`
	PayloadID     *PayloadID      `json:"payloadId"`
}

type ForkchoiceStateV1 struct {
	HeadBlockHash      string `json:"headBlockHash"`
	SafeBlockHash      string `json:"safeBlockHash"`
	FinalizedBlockHash string `json:"finalizedBlockHash"`
}

func encodeTransactions(txs [][]byte) [][]byte {
	enc := make([][]byte, len(txs))
	for i := range txs {
		if txs[i] == nil {
			continue
		}
		enc[i] = append([]byte(nil), txs[i]...)
	}
	return enc
}

func DecodeTransactions(enc [][]byte) ([][]byte, error) {
	txs := make([][]byte, len(enc))
	for i := range enc {
		if enc[i] == nil {
			continue
		}
		txs[i] = append([]byte(nil), enc[i]...)
	}
	return txs, nil
}

func ExecutableDataToBlock(data ExecutableData) (*ExecutableData, error) {
	block, err := ExecutableDataToBlockNoHash(data)
	if err != nil {
		return nil, err
	}
	if block.BlockHash != data.BlockHash {
		return nil, fmt.Errorf("blockhash mismatch, want %s, got %s", data.BlockHash, block.BlockHash)
	}
	return block, nil
}

func ExecutableDataToBlockNoHash(data ExecutableData) (*ExecutableData, error) {
	if len(data.LogsBloom) > 0 && len(data.LogsBloom) != 256 {
		return nil, fmt.Errorf("invalid logsBloom length: %v", len(data.LogsBloom))
	}
	copyData := data
	copyData.ExtraData = append([]byte(nil), data.ExtraData...)
	copyData.LogsBloom = append([]byte(nil), data.LogsBloom...)
	copyData.Transactions = encodeTransactions(data.Transactions)
	if data.Withdrawals != nil {
		copyData.Withdrawals = append([]string(nil), data.Withdrawals...)
	}
	return &copyData, nil
}

func BlockToExecutableData(block *ExecutableData, fees uint64, bundle *BlobsBundle, requests [][]byte) *ExecutionPayloadEnvelope {
	if block == nil {
		return &ExecutionPayloadEnvelope{
			ExecutionPayload: nil,
			BlockValue:       fees,
			BlobsBundle:      bundle,
			Requests:         cloneRequests(requests),
			Override:         false,
		}
	}

	copyData := *block
	copyData.ExtraData = append([]byte(nil), block.ExtraData...)
	copyData.LogsBloom = append([]byte(nil), block.LogsBloom...)
	copyData.Transactions = encodeTransactions(block.Transactions)
	if block.Withdrawals != nil {
		copyData.Withdrawals = append([]string(nil), block.Withdrawals...)
	}

	return &ExecutionPayloadEnvelope{
		ExecutionPayload: &copyData,
		BlockValue:       fees,
		BlobsBundle:      cloneBlobsBundle(bundle),
		Requests:         cloneRequests(requests),
		Override:         false,
	}
}

func cloneRequests(requests [][]byte) [][]byte {
	if requests == nil {
		return nil
	}
	out := make([][]byte, len(requests))
	for i := range requests {
		if requests[i] == nil {
			continue
		}
		out[i] = append([]byte(nil), requests[i]...)
	}
	return out
}

func cloneBlobsBundle(bundle *BlobsBundle) *BlobsBundle {
	if bundle == nil {
		return nil
	}
	out := &BlobsBundle{
		Commitments: make([][]byte, len(bundle.Commitments)),
		Proofs:      make([][]byte, len(bundle.Proofs)),
		Blobs:       make([][]byte, len(bundle.Blobs)),
	}
	for i := range bundle.Commitments {
		if bundle.Commitments[i] != nil {
			out.Commitments[i] = append([]byte(nil), bundle.Commitments[i]...)
		}
	}
	for i := range bundle.Proofs {
		if bundle.Proofs[i] != nil {
			out.Proofs[i] = append([]byte(nil), bundle.Proofs[i]...)
		}
	}
	for i := range bundle.Blobs {
		if bundle.Blobs[i] != nil {
			out.Blobs[i] = append([]byte(nil), bundle.Blobs[i]...)
		}
	}
	return out
}

type ExecutionPayloadBody struct {
	TransactionData [][]byte `json:"transactions"`
	Withdrawals     []string `json:"withdrawals"`
}

const (
	ClientCode = "SI"
	ClientName = "sila"
)

type ClientVersionV1 struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

func (v *ClientVersionV1) String() string {
	return fmt.Sprintf("%s-%s-%s-%s", v.Code, v.Name, v.Version, v.Commit)
}
