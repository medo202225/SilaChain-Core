package core

type AccessTuple struct {
	Address     string
	StorageKeys []string
}

type Message struct {
	From                  string
	To                    *string
	Nonce                 uint64
	Value                 uint64
	GasLimit              uint64
	GasPrice              uint64
	GasFeeCap             uint64
	GasTipCap             uint64
	Data                  []byte
	AccessList            []AccessTuple
	BlobGasFeeCap         uint64
	BlobHashes            []string
	SkipNonceChecks       bool
	SkipTransactionChecks bool
	SkipAccountLoad       bool
}

func (m Message) ToAddress() string {
	if m.To == nil {
		return ""
	}
	return *m.To
}

func (m Message) EffectiveGasPrice() uint64 {
	if m.GasPrice > 0 {
		return m.GasPrice
	}
	if m.GasFeeCap > 0 {
		return m.GasFeeCap
	}
	return 0
}
