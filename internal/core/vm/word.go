package vm

import "math/big"

var (
	twoTo256 = new(big.Int).Lsh(big.NewInt(1), 256)
	wordMask = new(big.Int).Sub(new(big.Int).Set(twoTo256), big.NewInt(1))
)

func NormalizeWord(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}

	out := new(big.Int).Set(v)
	out.And(out, wordMask)
	return out
}

func NewWordFromUint64(v uint64) *big.Int {
	return new(big.Int).SetUint64(v)
}

func NewWordFromBytes(b []byte) *big.Int {
	if len(b) == 0 {
		return new(big.Int)
	}
	return NormalizeWord(new(big.Int).SetBytes(b))
}

func WordToBytes32(v *big.Int) []byte {
	n := NormalizeWord(v)
	raw := n.Bytes()
	if len(raw) >= 32 {
		if len(raw) == 32 {
			return raw
		}
		return raw[len(raw)-32:]
	}

	out := make([]byte, 32)
	copy(out[32-len(raw):], raw)
	return out
}
