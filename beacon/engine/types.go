package engine

import (
	"encoding/hex"
	"fmt"
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
	Transactions  []string `json:"transactions"`
	Withdrawals   []string `json:"withdrawals,omitempty"`
	BlobGasUsed   *uint64  `json:"blobGasUsed,omitempty"`
	ExcessBlobGas *uint64  `json:"excessBlobGas,omitempty"`
	SlotNumber    *uint64  `json:"slotNumber,omitempty"`
}

type StatelessPayloadStatusV1 struct {
	Status          string  `json:"status"`
	StateRoot       string  `json:"stateRoot"`
	ReceiptsRoot    string  `json:"receiptsRoot"`
	ValidationError *string `json:"validationError"`
}

type BlobsBundle struct {
	Commitments [][]byte `json:"commitments"`
	Proofs      [][]byte `json:"proofs"`
	Blobs       [][]byte `json:"blobs"`
}

type ExecutionPayloadEnvelope struct {
	ExecutionPayload *ExecutableData `json:"executionPayload"`
	BlockValue       uint64          `json:"blockValue"`
	BlobsBundle      *BlobsBundle    `json:"blobsBundle,omitempty"`
	Requests         [][]byte        `json:"executionRequests,omitempty"`
	Override         bool            `json:"shouldOverrideBuilder"`
	Witness          *[]byte         `json:"witness,omitempty"`
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
	for _, v := range versions {
		if b.Version() == v {
			return true
		}
	}
	return false
}

func (b PayloadID) String() string {
	return "0x" + hex.EncodeToString(b[:])
}

func (b PayloadID) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b *PayloadID) UnmarshalText(input []byte) error {
	if len(input) >= 2 && string(input[:2]) == "0x" {
		input = input[2:]
	}
	raw, err := hex.DecodeString(string(input))
	if err != nil {
		return fmt.Errorf("invalid payload id %q: %w", string(input), err)
	}
	if len(raw) != len(b[:]) {
		return fmt.Errorf("invalid payload id length: %d", len(raw))
	}
	copy(b[:], raw)
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

func encodeTransactions(txs []string) []string {
	out := make([]string, len(txs))
	copy(out, txs)
	return out
}

func DecodeTransactions(enc []string) ([]string, error) {
	out := make([]string, len(enc))
	copy(out, enc)
	return out, nil
}

func ExecutableDataToBlock(data ExecutableData) (*ExecutableData, error) {
	if data.BlockHash == "" {
		return nil, fmt.Errorf("blockhash mismatch, want non-empty hash")
	}
	copyData := data
	copyData.Transactions = encodeTransactions(data.Transactions)
	return &copyData, nil
}

func ExecutableDataToBlockNoHash(data ExecutableData) (*ExecutableData, error) {
	copyData := data
	copyData.Transactions = encodeTransactions(data.Transactions)
	return &copyData, nil
}

func BlockToExecutableData(data ExecutableData, fees uint64, bundle *BlobsBundle, requests [][]byte) *ExecutionPayloadEnvelope {
	copyData := data
	copyData.Transactions = encodeTransactions(data.Transactions)

	return &ExecutionPayloadEnvelope{
		ExecutionPayload: &copyData,
		BlockValue:       fees,
		BlobsBundle:      bundle,
		Requests:         requests,
		Override:         false,
	}
}

type ExecutionPayloadBody struct {
	TransactionData []string `json:"transactions"`
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
