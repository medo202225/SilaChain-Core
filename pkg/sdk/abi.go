package sdk

import "silachain/internal/core/vm"

type ABIArgument = vm.ABIArgument
type ABIType = vm.ABIType

const (
	ABITypeUint256 = vm.ABITypeUint256
	ABITypeAddress = vm.ABITypeAddress
	ABITypeBytes32 = vm.ABITypeBytes32
	ABITypeBool    = vm.ABITypeBool
)

func EncodeCallData(signature string, args ...ABIArgument) (string, error) {
	return vm.EncodeABICallHex(signature, args...)
}

func FunctionSelectorHex(signature string) string {
	return vm.FunctionSelectorHex(signature)
}
