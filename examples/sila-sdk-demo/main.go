package main

import (
	"encoding/json"
	"fmt"
	"os"

	"silachain/pkg/sdk"
)

func main() {
	calldata, err := sdk.EncodeCallData(
		"transfer(address,uint256)",
		sdk.ABIArgument{Type: sdk.ABITypeAddress, Value: "0x1111111111111111111111111111111111111111"},
		sdk.ABIArgument{Type: sdk.ABITypeUint256, Value: uint64(5)},
	)
	if err != nil {
		fmt.Println("encode error:", err)
		os.Exit(1)
	}

	call := sdk.NewReadOnlyCall(sdk.ReadOnlyCallRequest{
		To:    "SILA_CONTRACT_001",
		Input: calldata,
	})

	out, err := json.MarshalIndent(call, "", "  ")
	if err != nil {
		fmt.Println("json error:", err)
		os.Exit(1)
	}

	fmt.Println(string(out))
}
