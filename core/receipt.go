package core

type Receipt struct {
	TxHash       string
	From         string
	To           string
	Nonce        uint64
	GasUsed      uint64
	RefundedGas  uint64
	RemainingGas uint64
	Success      bool
	ErrorText    string
	ReturnData   []byte
}

func (r Receipt) Failed() bool {
	return !r.Success
}

type Result struct {
	BlockNumber        uint64
	BlockHash          string
	ParentHash         string
	ExecutionStateRoot string
	BaseFee            uint64
	GasUsed            uint64
	Receipts           []Receipt
	TxCount            int
	SuccessCount       int
	FailureCount       int
}
