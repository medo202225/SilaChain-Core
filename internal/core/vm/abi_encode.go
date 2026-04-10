package vm

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var (
	ErrUnsupportedABIType = errors.New("vm: unsupported abi type")
	ErrInvalidABIValue    = errors.New("vm: invalid abi value")
)

func EncodeABICall(signature string, args ...ABIArgument) ([]byte, error) {
	selector := FunctionSelector(signature)
	if len(selector) != FunctionSelectorSize {
		return nil, ErrInvalidABIValue
	}

	out := make([]byte, 0, FunctionSelectorSize+(32*len(args)))
	out = append(out, selector...)

	for _, arg := range args {
		encoded, err := EncodeABIArgument(arg)
		if err != nil {
			return nil, err
		}
		out = append(out, encoded...)
	}

	return out, nil
}

func EncodeABIArgument(arg ABIArgument) ([]byte, error) {
	switch arg.Type {
	case ABITypeUint256:
		return encodeABIUint256(arg.Value)
	case ABITypeAddress:
		return encodeABIAddress(arg.Value)
	case ABITypeBytes32:
		return encodeABIBytes32(arg.Value)
	case ABITypeBool:
		return encodeABIBool(arg.Value)
	default:
		return nil, ErrUnsupportedABIType
	}
}

func EncodeABICallHex(signature string, args ...ABIArgument) (string, error) {
	encoded, err := EncodeABICall(signature, args...)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(encoded), nil
}

func encodeABIUint256(v any) ([]byte, error) {
	n, err := normalizeABIInteger(v)
	if err != nil {
		return nil, err
	}
	if n.Sign() < 0 {
		return nil, ErrInvalidABIValue
	}
	return WordToBytes32(n), nil
}

func encodeABIAddress(v any) ([]byte, error) {
	raw, ok := v.(string)
	if !ok {
		return nil, ErrInvalidABIValue
	}

	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")

	if len(s) > 40 {
		return nil, ErrInvalidABIValue
	}
	if len(s)%2 != 0 {
		s = "0" + s
	}

	decoded, err := hex.DecodeString(s)
	if err != nil {
		return nil, ErrInvalidABIValue
	}
	if len(decoded) > 20 {
		return nil, ErrInvalidABIValue
	}

	out := make([]byte, 32)
	copy(out[32-len(decoded):], decoded)
	return out, nil
}

func encodeABIBytes32(v any) ([]byte, error) {
	switch t := v.(type) {
	case []byte:
		if len(t) > 32 {
			return nil, ErrInvalidABIValue
		}
		out := make([]byte, 32)
		copy(out, t)
		return out, nil
	case string:
		s := strings.TrimSpace(t)
		s = strings.TrimPrefix(s, "0x")
		s = strings.TrimPrefix(s, "0X")

		if s == "" {
			return make([]byte, 32), nil
		}
		if len(s)%2 != 0 {
			s = "0" + s
		}

		decoded, err := hex.DecodeString(s)
		if err != nil {
			return nil, ErrInvalidABIValue
		}
		if len(decoded) > 32 {
			return nil, ErrInvalidABIValue
		}

		out := make([]byte, 32)
		copy(out, decoded)
		return out, nil
	default:
		return nil, ErrInvalidABIValue
	}
}

func encodeABIBool(v any) ([]byte, error) {
	b, ok := v.(bool)
	if !ok {
		return nil, ErrInvalidABIValue
	}
	if b {
		return WordToBytes32(NewWordFromUint64(1)), nil
	}
	return WordToBytes32(NewWordFromUint64(0)), nil
}

func normalizeABIInteger(v any) (*big.Int, error) {
	switch t := v.(type) {
	case uint8:
		return new(big.Int).SetUint64(uint64(t)), nil
	case uint16:
		return new(big.Int).SetUint64(uint64(t)), nil
	case uint32:
		return new(big.Int).SetUint64(uint64(t)), nil
	case uint64:
		return new(big.Int).SetUint64(t), nil
	case uint:
		return new(big.Int).SetUint64(uint64(t)), nil
	case int8:
		return big.NewInt(int64(t)), nil
	case int16:
		return big.NewInt(int64(t)), nil
	case int32:
		return big.NewInt(int64(t)), nil
	case int64:
		return big.NewInt(t), nil
	case int:
		return big.NewInt(int64(t)), nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil, ErrInvalidABIValue
		}
		n, ok := new(big.Int).SetString(s, 10)
		if !ok {
			return nil, fmt.Errorf("%w: uint256 string", ErrInvalidABIValue)
		}
		return n, nil
	case *big.Int:
		if t == nil {
			return nil, ErrInvalidABIValue
		}
		return new(big.Int).Set(t), nil
	default:
		return nil, ErrInvalidABIValue
	}
}
