package execution

type ForkchoiceStateV1 struct {
	HeadBlockHash      string `json:"headBlockHash"`
	SafeBlockHash      string `json:"safeBlockHash"`
	FinalizedBlockHash string `json:"finalizedBlockHash"`
}

type PayloadAttributesV1 struct {
	Timestamp             string `json:"timestamp"`
	PrevRandao            string `json:"prevRandao"`
	SuggestedFeeRecipient string `json:"suggestedFeeRecipient"`
}

type PayloadStatusV1 struct {
	Status          string  `json:"status"`
	LatestValidHash *string `json:"latestValidHash"`
	ValidationError *string `json:"validationError"`
}

type ForkchoiceUpdatedResponseV1 struct {
	PayloadStatus PayloadStatusV1 `json:"payloadStatus"`
	PayloadID     *string         `json:"payloadId"`
}

type ExecutionPayloadV1 struct {
	BlockHash    string `json:"blockHash"`
	ParentHash   string `json:"parentHash"`
	BlockNumber  string `json:"blockNumber"`
	Timestamp    string `json:"timestamp"`
	PrevRandao   string `json:"prevRandao"`
	FeeRecipient string `json:"feeRecipient"`
}

type GetPayloadResponseV1 struct {
	ExecutionPayload ExecutionPayloadV1 `json:"executionPayload"`
	BlockValue       string             `json:"blockValue"`
}
