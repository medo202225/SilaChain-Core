package core

type BlockContext struct {
	BlockNumber uint64
	BlockHash   string
	ParentHash  string
	BaseFee     uint64
	GasLimit    uint64
	Timestamp   uint64
}

func NewBlockContext(
	blockNumber uint64,
	blockHash string,
	parentHash string,
	baseFee uint64,
	gasLimit uint64,
	timestamp uint64,
) BlockContext {
	return BlockContext{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
		ParentHash:  parentHash,
		BaseFee:     baseFee,
		GasLimit:    gasLimit,
		Timestamp:   timestamp,
	}
}
