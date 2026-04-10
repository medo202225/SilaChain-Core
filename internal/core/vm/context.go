package vm

type BlockContext struct {
	ChainID       string
	BlockNumber   uint64
	BlockTimeUnix int64
	Proposer      string
	BaseFee       uint64
	GasLimit      uint64
}

type TxContext struct {
	TxHash   string
	Origin   string
	GasPrice uint64
}

type ExecutionContext struct {
	VMVersion    uint16
	ContractAddr string
	CodeAddr     string
	StorageAddr  string
	Caller       string
	Origin       string
	CallValue    uint64
	Input        []byte
	GasRemaining uint64
	Depth        uint16
	Static       bool
	Block        BlockContext
	Tx           TxContext
}

func (c ExecutionContext) Clone() ExecutionContext {
	out := c
	out.Input = cloneBytes(c.Input)
	return out
}
