package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"silachain/pkg/sdk"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage:")
		fmt.Println("  sila-wallet new")
		fmt.Println("  sila-wallet import <private_key_hex>")
		fmt.Println("  sila-wallet calldata <signature>")
		fmt.Println("  sila-wallet receipt <base_url> <tx_hash>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		w, err := sdk.NewWallet()
		if err != nil {
			fmt.Println("wallet error:", err)
			os.Exit(1)
		}
		out, _ := json.MarshalIndent(w, "", "  ")
		fmt.Println(string(out))

	case "import":
		if len(os.Args) < 3 {
			fmt.Println("missing private key")
			os.Exit(1)
		}
		w, err := sdk.ImportWallet(os.Args[2])
		if err != nil {
			fmt.Println("import error:", err)
			os.Exit(1)
		}
		out, _ := json.MarshalIndent(w, "", "  ")
		fmt.Println(string(out))

	case "calldata":
		if len(os.Args) < 3 {
			fmt.Println("missing signature")
			os.Exit(1)
		}
		fmt.Println(sdk.FunctionSelectorHex(os.Args[2]))

	case "receipt":
		if len(os.Args) < 4 {
			fmt.Println("missing base_url or tx_hash")
			os.Exit(1)
		}
		client := sdk.NewClient(os.Args[2])
		receipt, err := client.WaitForReceipt(os.Args[3], 3, 2*time.Second)
		if err != nil {
			fmt.Println("receipt error:", err)
			os.Exit(1)
		}
		out, _ := json.MarshalIndent(receipt, "", "  ")
		fmt.Println(string(out))

	default:
		fmt.Println("unknown command")
		os.Exit(1)
	}
}
