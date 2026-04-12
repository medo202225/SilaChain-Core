package state

type StateLog struct {
	Address string
	Topics  []string
	Data    []byte
}

type Reader interface {
	GetBalance(address string) uint64
	GetNonce(address string) uint64
	GetState(address, key string) (string, bool)
	GetCode(address string) []byte
	GetCodeHash(address string) string
	Exist(address string) bool
	Empty(address string) bool
}
