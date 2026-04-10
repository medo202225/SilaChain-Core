package vm

type CallResult struct {
	Success    bool
	ReturnData []byte
	RevertData []byte
	GasUsed    uint64
	Err        error
}

type CreateResult struct {
	Success         bool
	ContractAddress string
	RuntimeCode     []byte
	GasUsed         uint64
	Err             error
}

type Host interface {
	AccountExists(address string) bool
	GetBalance(address string) uint64
	Transfer(from, to string, amount uint64) error

	GetCode(address string) []byte
	SetCode(address string, code []byte) error
	DeleteCode(address string) error

	GetStorage(address, key string) []byte
	SetStorage(address, key string, value []byte) error

	EmitLog(entry LogEntry)

	CreateCheckpoint() int
	CommitCheckpoint(id int) error
	RevertCheckpoint(id int) error

	CallContract(caller string, target string, input []byte, value uint64, gas uint64, static bool) CallResult
	CreateContractAddress(caller string) (string, error)
}
