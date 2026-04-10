package consensuslegacy

type Checkpoint struct {
	Epoch     Epoch  `json:"epoch"`
	BlockHash string `json:"block_hash"`
}

func NewCheckpoint(epoch Epoch, blockHash string) Checkpoint {
	return Checkpoint{
		Epoch:     epoch,
		BlockHash: blockHash,
	}
}
