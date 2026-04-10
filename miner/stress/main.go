package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/txpool"
	"silachain/miner"
)

type faucet struct {
	Address string
	Nonce   uint64
}

func main() {
	faucets := make([]faucet, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i] = faucet{
			Address: fmt.Sprintf("SILA_FAUCET_%03d", i),
			Nonce:   0,
		}
	}

	interruptCh := make(chan os.Signal, 5)
	signal.Notify(interruptCh, os.Interrupt)

	m := miner.New(nil, miner.Config{
		PendingFeeRecipient: "SILA_STRESS_FEE_RECIPIENT",
		GasCeil:             25_000_000,
		Recommit:            2 * time.Second,
	})

	sent := uint64(0)

	for {
		select {
		case <-interruptCh:
			return
		default:
		}

		index := rand.Intn(len(faucets))
		faucets[index].Nonce++

		tx := txpool.Tx{
			Hash:                 makeTxHash(faucets[index].Address, faucets[index].Nonce, sent),
			From:                 faucets[index].Address,
			Nonce:                faucets[index].Nonce - 1,
			GasLimit:             21000,
			MaxFeePerGas:         100,
			MaxPriorityFeePerGas: 2,
			Timestamp:            time.Now().Unix(),
		}

		args := miner.BuildPayloadArgs{
			ParentHash:   fmt.Sprintf("sila-stress-parent-%d", sent/16),
			Timestamp:    uint64(time.Now().Unix()),
			FeeRecipient: "SILA_STRESS_FEE_RECIPIENT",
			Random:       fmt.Sprintf("SILA_STRESS_RANDOM_%d", sent),
			GasLimit:     25_000_000,
			Version:      1,
		}

		built := blockassembly.Result{
			ParentNumber:    sent / 16,
			BlockNumber:     sent/16 + 1,
			ParentHash:      args.ParentHash,
			ParentStateRoot: fmt.Sprintf("sila-stress-parent-state-%d", sent/16),
			BaseFee:         1,
			GasLimit:        25_000_000,
			Attributes: blockassembly.PayloadAttributes{
				Timestamp:         args.Timestamp,
				FeeRecipient:      args.FeeRecipient,
				Random:            args.Random,
				SuggestedGasLimit: args.GasLimit,
			},
			Selection: blockassembly.TransactionSelection{
				Transactions: []txpool.Tx{tx},
				GasUsed:      21000,
				TotalTipFees: tx.EffectiveFee(1),
			},
		}

		payload, err := m.BuildPayload(context.Background(), args, built, fmt.Sprintf("sila-stress-state-%d", sent))
		if err != nil {
			panic(err)
		}
		_ = payload.Resolve()

		sent++

		if sent%256 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func makeTxHash(address string, nonce uint64, sent uint64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d", address, nonce, sent)))
	return "0x" + hex.EncodeToString(sum[:])
}
