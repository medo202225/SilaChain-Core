package core

import (
	"testing"

	statecore "silachain/core/state"
)

func TestApplyMessageTransfersValueAndAdvancesNonce(t *testing.T) {
	db := statecore.NewStateDB()
	db.SetBalance("SILA_SENDER", 1000000)
	db.SetNonce("SILA_SENDER", 0)

	to := "SILA_RECEIVER"
	gp := NewGasPool(50000)

	result, err := ApplyMessage(db, gp, Message{
		From:                  "SILA_SENDER",
		To:                    &to,
		Nonce:                 0,
		Value:                 5,
		GasLimit:              21000,
		GasPrice:              7,
		GasFeeCap:             7,
		SkipNonceChecks:       false,
		SkipTransactionChecks: false,
		SkipAccountLoad:       false,
		Data:                  nil,
	})
	if err != nil {
		t.Fatalf("apply message: %v", err)
	}

	if result.Failed() {
		t.Fatalf("expected successful result")
	}
	if result.PurchaseGas != 21000 {
		t.Fatalf("purchase gas = %d", result.PurchaseGas)
	}
	if result.PurchaseCost != 147000 {
		t.Fatalf("purchase cost = %d", result.PurchaseCost)
	}
	if result.UsedGas != 21000 {
		t.Fatalf("used gas = %d", result.UsedGas)
	}
	if result.MaxUsedGas != 21000 {
		t.Fatalf("max used gas = %d", result.MaxUsedGas)
	}
	if result.RemainingGas != 0 {
		t.Fatalf("remaining gas = %d", result.RemainingGas)
	}
	if result.RefundedGas != 0 {
		t.Fatalf("refunded gas = %d", result.RefundedGas)
	}
	if got := db.GetNonce("SILA_SENDER"); got != 1 {
		t.Fatalf("sender nonce = %d", got)
	}
	if got := db.GetBalance("SILA_RECEIVER"); got != 5 {
		t.Fatalf("receiver balance = %d", got)
	}

	wantSenderBalance := uint64(1000000 - 5 - 21000*7)
	if got := db.GetBalance("SILA_SENDER"); got != wantSenderBalance {
		t.Fatalf("sender balance = %d want=%d", got, wantSenderBalance)
	}
	if got := gp.Gas(); got != 29000 {
		t.Fatalf("gas pool = %d", got)
	}
}

func TestApplyMessageFailsOnNonceMismatch(t *testing.T) {
	db := statecore.NewStateDB()
	db.SetBalance("alice", 1000000)
	db.SetNonce("alice", 1)

	to := "sink"
	gp := NewGasPool(50000)

	result, err := ApplyMessage(db, gp, Message{
		From:                  "alice",
		To:                    &to,
		Nonce:                 0,
		Value:                 0,
		GasLimit:              21000,
		GasPrice:              1,
		GasFeeCap:             1,
		SkipNonceChecks:       false,
		SkipTransactionChecks: false,
		SkipAccountLoad:       false,
	})
	if err == nil {
		t.Fatalf("expected nonce mismatch error")
	}
	if !result.Failed() {
		t.Fatalf("expected failed result")
	}
}

func TestApplyMessageWritesCalldataAndRefundsRemainingGas(t *testing.T) {
	db := statecore.NewStateDB()
	db.SetBalance("alice", 1000000)
	db.SetNonce("alice", 0)

	to := "contract1"
	gp := NewGasPool(50000)

	msg := Message{
		From:                  "alice",
		To:                    &to,
		Nonce:                 0,
		Value:                 0,
		GasLimit:              25000,
		GasPrice:              2,
		GasFeeCap:             2,
		SkipNonceChecks:       false,
		SkipTransactionChecks: false,
		SkipAccountLoad:       false,
		Data:                  []byte("abcd"),
	}

	result, err := ApplyMessage(db, gp, msg)
	if err != nil {
		t.Fatalf("apply message: %v", err)
	}

	wantUsed := uint64(21000 + 4*16)
	if result.UsedGas != wantUsed {
		t.Fatalf("used gas = %d want=%d", result.UsedGas, wantUsed)
	}
	wantRemaining := uint64(25000) - wantUsed
	if result.RemainingGas != wantRemaining {
		t.Fatalf("remaining gas = %d want=%d", result.RemainingGas, wantRemaining)
	}
	if result.RefundedGas != wantRemaining {
		t.Fatalf("refunded gas = %d want=%d", result.RefundedGas, wantRemaining)
	}
	if result.PurchaseGas != 25000 {
		t.Fatalf("purchase gas = %d", result.PurchaseGas)
	}
	if result.PurchaseCost != 50000 {
		t.Fatalf("purchase cost = %d", result.PurchaseCost)
	}
	if string(result.Return()) != "abcd" {
		t.Fatalf("return data = %q", string(result.Return()))
	}

	value, ok := db.GetState("contract1", "calldata")
	if !ok {
		t.Fatalf("expected calldata state")
	}
	if value != "abcd" {
		t.Fatalf("calldata = %q", value)
	}
}

func TestApplyMessageFailsOnInsufficientFunds(t *testing.T) {
	db := statecore.NewStateDB()
	db.SetBalance("poor", 100)
	db.SetNonce("poor", 0)

	to := "sink"
	gp := NewGasPool(50000)

	result, err := ApplyMessage(db, gp, Message{
		From:                  "poor",
		To:                    &to,
		Nonce:                 0,
		Value:                 0,
		GasLimit:              21000,
		GasPrice:              1,
		GasFeeCap:             1,
		SkipNonceChecks:       false,
		SkipTransactionChecks: false,
		SkipAccountLoad:       false,
	})
	if err == nil {
		t.Fatalf("expected insufficient funds error")
	}
	if !result.Failed() {
		t.Fatalf("expected failed result")
	}
}

func TestApplyTransactionBuildsReceipt(t *testing.T) {
	db := statecore.NewStateDB()
	db.SetBalance("alice", 1000000)
	db.SetNonce("alice", 0)

	to := "sink"
	gp := NewGasPool(50000)
	ctx := BlockContext{
		BlockNumber: 1,
		BlockHash:   "0x1",
		ParentHash:  "0x0",
		BaseFee:     1,
		GasLimit:    50000,
		Timestamp:   1,
	}

	receipt, err := ApplyTransaction(ctx, db, gp, "tx-1", Message{
		From:                  "alice",
		To:                    &to,
		Nonce:                 0,
		Value:                 0,
		GasLimit:              21000,
		GasPrice:              1,
		GasFeeCap:             1,
		SkipNonceChecks:       false,
		SkipTransactionChecks: false,
		SkipAccountLoad:       false,
	})
	if err != nil {
		t.Fatalf("apply transaction: %v", err)
	}
	if receipt.TxHash != "tx-1" {
		t.Fatalf("tx hash = %s", receipt.TxHash)
	}
	if !receipt.Success {
		t.Fatalf("expected success")
	}
}
