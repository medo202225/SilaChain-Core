package vm

type ABIType string

const (
	ABITypeUint256 ABIType = "uint256"
	ABITypeAddress ABIType = "address"
	ABITypeBytes32 ABIType = "bytes32"
	ABITypeBool    ABIType = "bool"
)

type ABIArgument struct {
	Type  ABIType
	Value any
}
