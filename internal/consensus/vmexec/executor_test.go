package vmexec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"silachain/internal/consensus/vmstate"
)

func word32FromUint64(v uint64) []byte {
	out := make([]byte, 32)
	binary.BigEndian.PutUint64(out[24:], v)
	return out
}

func concatBytes(parts ...[]byte) []byte {
	var out []byte
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func trailingUint64(data []byte) uint64 {
	if len(data) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(data[len(data)-8:])
}

func jumpDestIndex(code []byte) byte {
	for i, op := range code {
		if op == OpJUMPDEST {
			return byte(i)
		}
	}
	panic("missing JUMPDEST in test bytecode")
}

func TestExecutor_Execute_NoCodeAccount(t *testing.T) {
	st := vmstate.New()
	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "bob",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !result.Success || result.CodeExecuted || !result.CreatedAccount {
		t.Fatalf("unexpected no-code result")
	}
	if result.GasUsed != 21000 {
		t.Fatalf("unexpected gas used: got=%d want=21000", result.GasUsed)
	}
	if len(result.Logs) != 0 {
		t.Fatalf("unexpected no-code logs count: got=%d want=0", len(result.Logs))
	}
}

func TestExecutor_Execute_EQ_ReturnsOne(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPUSH1, 0x05,
		OpPUSH1, 0x05,
		OpEQ,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}
	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, err := exec.Execute(ExecutionContext{}, Message{From: "alice", To: "contract1", GasLimit: 50000})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	want := word32FromUint64(1)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_LT_ReturnsOne(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPUSH1, 0x03,
		OpPUSH1, 0x05,
		OpLT,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}
	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, err := exec.Execute(ExecutionContext{}, Message{From: "alice", To: "contract1", GasLimit: 50000})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	want := word32FromUint64(1)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_GT_ReturnsOne(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPUSH1, 0x05,
		OpPUSH1, 0x03,
		OpGT,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}
	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, err := exec.Execute(ExecutionContext{}, Message{From: "alice", To: "contract1", GasLimit: 50000})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	want := word32FromUint64(1)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_ISZERO_ReturnsOne(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPUSH1, 0x00,
		OpISZERO,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}
	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, err := exec.Execute(ExecutionContext{}, Message{From: "alice", To: "contract1", GasLimit: 50000})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	want := word32FromUint64(1)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_JUMPI_UsingComparison(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpPUSH1, 0x05,
		OpPUSH1, 0x05,
		OpEQ,
		OpPUSH1, 0x00,
		OpJUMPI,
		OpSTOP,
		OpSTOP,
		OpJUMPDEST,
		OpPUSH1, 0x2b,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	code[6] = jumpDestIndex(code)

	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}
	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, err := exec.Execute(ExecutionContext{}, Message{From: "alice", To: "contract1", GasLimit: 50000})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	want := word32FromUint64(0x2b)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_LOG0_EmitsLog(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPUSH1, 0x2a,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpLOG0,
		OpSTOP,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(result.Logs) != 1 {
		t.Fatalf("unexpected logs count: got=%d want=1", len(result.Logs))
	}
	if result.Logs[0].Address != "contract1" {
		t.Fatalf("unexpected log address: got=%s want=contract1", result.Logs[0].Address)
	}
	if len(result.Logs[0].Topics) != 0 {
		t.Fatalf("unexpected topic count: got=%d want=0", len(result.Logs[0].Topics))
	}

	wantData := word32FromUint64(0x2a)
	if !bytes.Equal(result.Logs[0].Data, wantData) {
		t.Fatalf("unexpected log data: got=%v want=%v", result.Logs[0].Data, wantData)
	}
}

func TestExecutor_Execute_LOG1_EmitsLogWithTopic(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPUSH1, 0x7b,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpPUSH1, 0x99,
		OpLOG1,
		OpSTOP,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(result.Logs) != 1 {
		t.Fatalf("unexpected logs count: got=%d want=1", len(result.Logs))
	}
	if result.Logs[0].Address != "contract1" {
		t.Fatalf("unexpected log address: got=%s want=contract1", result.Logs[0].Address)
	}
	if len(result.Logs[0].Topics) != 1 {
		t.Fatalf("unexpected topic count: got=%d want=1", len(result.Logs[0].Topics))
	}
	if result.Logs[0].Topics[0] != "0x99" {
		t.Fatalf("unexpected topic value: got=%s want=0x99", result.Logs[0].Topics[0])
	}

	wantData := word32FromUint64(0x7b)
	if !bytes.Equal(result.Logs[0].Data, wantData) {
		t.Fatalf("unexpected log data: got=%v want=%v", result.Logs[0].Data, wantData)
	}
}

func TestExecutor_Execute_CALL_ReturnsSuccessAndCopiesReturnData(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x4d,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpCALL,

		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !result.Success || result.Reverted {
		t.Fatalf("unexpected call status: success=%v reverted=%v", result.Success, result.Reverted)
	}

	want := word32FromUint64(0x4d)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CALL_PropagatesCalleeLogs(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x6a,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x08,
		OpPUSH1, 0xaa,
		OpLOG1,
		OpSTOP,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x08,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpCALL,
		OpSTOP,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(result.Logs) != 1 {
		t.Fatalf("unexpected logs count: got=%d want=1", len(result.Logs))
	}
	if result.Logs[0].Address != "contract2" {
		t.Fatalf("unexpected propagated log address: got=%s want=contract2", result.Logs[0].Address)
	}
	if len(result.Logs[0].Topics) != 1 || result.Logs[0].Topics[0] != "0xaa" {
		t.Fatalf("unexpected propagated log topics: got=%v", result.Logs[0].Topics)
	}
}

func TestExecutor_Execute_CALL_PassesCalldataToCallee(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x00,
		OpCALLDATALOAD,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x33,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpCALL,
		OpPUSH1, 0x20,
		OpMLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0x33)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected calldata return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CALL_CalleeRevertReturnsZeroAndCopiesRevertData(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x7e,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpREVERT,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpCALL,

		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x00),
		word32FromUint64(0x7e),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected revert return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CALL_ContextOpcodes(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpADDRESS,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpCALLER,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpCALLVALUE,
		OpPUSH1, 0x40,
		OpMSTORE,

		OpCALLDATASIZE,
		OpPUSH1, 0x60,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x80,
		OpRETURN,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x44,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x80,
		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x05,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpCALL,

		OpPUSH1, 0x20,
		OpPUSH1, 0x80,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x02),
		word32FromUint64(0x01),
		word32FromUint64(0x05),
		word32FromUint64(0x20),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected context return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CALL_CalleeStorageIsolation(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x55,
		OpPUSH1, 0x01,
		OpSSTORE,
		OpPUSH1, 0x01,
		OpSLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x99,
		OpPUSH1, 0x01,
		OpSSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH2, 0x08, 0x98,
		OpCALL,

		OpPUSH1, 0x20,
		OpMLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x01,
		OpSLOAD,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x55),
		word32FromUint64(0x99),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected storage isolation return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CALL_ToMissingAccountZerosReturnAreaAndSucceeds(t *testing.T) {
	st := vmstate.New()

	callerCode := []byte{
		OpPUSH1, 0xab,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x09,
		OpPUSH1, 0xff,
		OpCALL,

		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpMLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x01),
		word32FromUint64(0x01),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected missing-account return data: got=%v want=%v", result.ReturnData, want)
	}
	if len(result.Logs) != 0 {
		t.Fatalf("unexpected missing-account logs count: got=%d want=0", len(result.Logs))
	}
}

func TestExecutor_Execute_CALL_ToExistingNoCodeAccountZerosReturnAreaAndSucceeds(t *testing.T) {
	st := vmstate.New()
	if _, err := st.EnsureAccount("contract7"); err != nil {
		t.Fatalf("ensure account: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0xcd,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x07,
		OpPUSH1, 0xff,
		OpCALL,

		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpMLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x01),
		word32FromUint64(0x01),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected no-code return data: got=%v want=%v", result.ReturnData, want)
	}
	if len(result.Logs) != 0 {
		t.Fatalf("unexpected no-code logs count: got=%d want=0", len(result.Logs))
	}
}

func TestExecutor_Execute_CALL_DepthLimitReturnsZeroAndZerosRetArea(t *testing.T) {
	st := vmstate.New()

	leafCode := []byte{
		OpPUSH1, 0xee,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract6", leafCode); err != nil {
		t.Fatalf("set leaf code: %v", err)
	}

	reentrantTemplate := func(target byte) []byte {
		return []byte{
			OpPUSH1, 0xaa,
			OpPUSH1, 0x20,
			OpMSTORE,

			OpPUSH1, 0x20,
			OpPUSH1, 0x20,
			OpPUSH1, 0x00,
			OpPUSH1, 0x00,
			OpPUSH1, 0x00,
			OpPUSH1, target,
			OpPUSH1, 0xff,
			OpCALL,

			OpPUSH1, 0x20,
			OpMSTORE,

			OpPUSH1, 0x20,
			OpMLOAD,
			OpPUSH1, 0x00,
			OpMSTORE,

			OpPUSH1, 0x00,
			OpPUSH1, 0x40,
			OpRETURN,
		}
	}

	if err := st.SetCode("contract2", reentrantTemplate(0x03)); err != nil {
		t.Fatalf("set reentrant code for contract2: %v", err)
	}
	if err := st.SetCode("contract3", reentrantTemplate(0x04)); err != nil {
		t.Fatalf("set reentrant code for contract3: %v", err)
	}
	if err := st.SetCode("contract4", reentrantTemplate(0x05)); err != nil {
		t.Fatalf("set reentrant code for contract4: %v", err)
	}
	if err := st.SetCode("contract5", reentrantTemplate(0x06)); err != nil {
		t.Fatalf("set reentrant code for contract5: %v", err)
	}

	entryCode := []byte{
		OpPUSH1, 0x10,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpCALL,

		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpMLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", entryCode); err != nil {
		t.Fatalf("set entry code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x01),
		word32FromUint64(0x01),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected depth-limit return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_InvalidOpcode(t *testing.T) {
	st := vmstate.New()
	if err := st.SetCode("contract1", []byte{0xfe}); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err == nil {
		t.Fatalf("expected invalid opcode error")
	}
}

func TestExecutor_Execute_RETURNDATASIZE_RETURNDATACOPY_AfterSuccessfulCALL(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x4d,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpCALL,

		OpPUSH1, 0x20,
		OpMSTORE,

		OpRETURNDATASIZE,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURNDATACOPY,

		OpPUSH1, 0x00,
		OpPUSH1, 0x60,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x20),
		word32FromUint64(0x4d),
		word32FromUint64(0x00),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected returndata success payload: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_RETURNDATASIZE_RETURNDATACOPY_AfterRevertedCALL(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x7e,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpREVERT,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpCALL,

		OpPUSH1, 0x20,
		OpMSTORE,

		OpRETURNDATASIZE,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURNDATACOPY,

		OpPUSH1, 0x00,
		OpPUSH1, 0x60,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x20),
		word32FromUint64(0x7e),
		word32FromUint64(0x00),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected returndata revert payload: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CALL_FailureCanBeAbsorbedByCaller(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x2a,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpREVERT,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpCALL,

		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x00),
		word32FromUint64(0x2a),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected absorbed-failure return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CALL_FailureCanBePropagatedByCaller(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x7e,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpREVERT,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpCALL,

		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpRETURNDATACOPY,

		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpREVERT,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0x7e)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected propagated-failure return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_DELEGATECALL_UsesCallerStorageAndOriginalContext(t *testing.T) {
	st := vmstate.New()

	libraryCode := []byte{
		OpCALLER,
		OpPUSH1, 0x01,
		OpSSTORE,

		OpCALLVALUE,
		OpPUSH1, 0x02,
		OpSSTORE,

		OpADDRESS,
		OpPUSH1, 0x03,
		OpSSTORE,

		OpPUSH1, 0x01,
		OpSLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x02,
		OpSLOAD,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x03,
		OpSLOAD,
		OpPUSH1, 0x40,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x60,
		OpRETURN,
	}
	if err := st.SetCode("contract2", libraryCode); err != nil {
		t.Fatalf("set library code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x60,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH2, 0x20, 0x00,
		OpDELEGATECALL,

		OpPUSH1, 0x20,
		OpPUSH1, 0x60,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		Value:    0x07,
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x00),
		word32FromUint64(0x07),
		word32FromUint64(0x02),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected delegatecall return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CALL_ContrastsWithDelegatecallStorageAndContext(t *testing.T) {
	st := vmstate.New()

	libraryCode := []byte{
		OpCALLER,
		OpPUSH1, 0x01,
		OpSSTORE,

		OpCALLVALUE,
		OpPUSH1, 0x02,
		OpSSTORE,

		OpADDRESS,
		OpPUSH1, 0x03,
		OpSSTORE,

		OpPUSH1, 0x01,
		OpSLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x02,
		OpSLOAD,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x03,
		OpSLOAD,
		OpPUSH1, 0x40,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x60,
		OpRETURN,
	}
	if err := st.SetCode("contract2", libraryCode); err != nil {
		t.Fatalf("set library code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x44,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x60,
		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x05,
		OpPUSH1, 0x02,
		OpPUSH2, 0x20, 0x00,
		OpCALL,

		OpPUSH1, 0x20,
		OpPUSH1, 0x60,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		Value:    0x09,
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x01),
		word32FromUint64(0x05),
		word32FromUint64(0x02),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected call return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_DELEGATECALL_PassesCalldataAndCopiesReturnData(t *testing.T) {
	st := vmstate.New()

	libraryCode := []byte{
		OpPUSH1, 0x00,
		OpCALLDATALOAD,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract2", libraryCode); err != nil {
		t.Fatalf("set library code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x33,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpDELEGATECALL,

		OpPUSH1, 0x20,
		OpMSTORE,

		OpRETURNDATASIZE,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURNDATACOPY,

		OpPUSH1, 0x00,
		OpPUSH1, 0x60,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x20),
		word32FromUint64(0x33),
		word32FromUint64(0x00),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected delegatecall calldata/returndata payload: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_DELEGATECALL_RevertAndReturnDataBuffer(t *testing.T) {
	st := vmstate.New()

	libraryCode := []byte{
		OpPUSH1, 0x7e,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpREVERT,
	}
	if err := st.SetCode("contract2", libraryCode); err != nil {
		t.Fatalf("set library code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpDELEGATECALL,

		OpPUSH1, 0x20,
		OpMSTORE,

		OpRETURNDATASIZE,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURNDATACOPY,

		OpPUSH1, 0x00,
		OpPUSH1, 0x60,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x20),
		word32FromUint64(0x7e),
		word32FromUint64(0x00),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected delegatecall revert/returndata payload: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_DELEGATECALL_ToMissingAccountZerosReturnAreaAndSucceeds(t *testing.T) {
	st := vmstate.New()

	callerCode := []byte{
		OpPUSH1, 0xab,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x09,
		OpPUSH1, 0xff,
		OpDELEGATECALL,

		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpMLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x01),
		word32FromUint64(0x01),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected missing-account delegatecall data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_DELEGATECALL_ToExistingNoCodeAccountZerosReturnAreaAndSucceeds(t *testing.T) {
	st := vmstate.New()
	if _, err := st.EnsureAccount("contract7"); err != nil {
		t.Fatalf("ensure account: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0xcd,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x07,
		OpPUSH1, 0xff,
		OpDELEGATECALL,

		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpMLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x01),
		word32FromUint64(0x01),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected no-code delegatecall data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_DELEGATECALL_DepthLimitReturnsZeroAndZerosRetArea(t *testing.T) {
	st := vmstate.New()

	leafCode := []byte{
		OpPUSH1, 0xee,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract6", leafCode); err != nil {
		t.Fatalf("set leaf code: %v", err)
	}

	reentrantTemplate := func(target byte) []byte {
		return []byte{
			OpPUSH1, 0xaa,
			OpPUSH1, 0x20,
			OpMSTORE,

			OpPUSH1, 0x20,
			OpPUSH1, 0x20,
			OpPUSH1, 0x00,
			OpPUSH1, 0x00,
			OpPUSH1, target,
			OpPUSH1, 0xff,
			OpDELEGATECALL,

			OpPUSH1, 0x20,
			OpMSTORE,

			OpPUSH1, 0x20,
			OpMLOAD,
			OpPUSH1, 0x00,
			OpMSTORE,

			OpPUSH1, 0x00,
			OpPUSH1, 0x40,
			OpRETURN,
		}
	}

	if err := st.SetCode("contract2", reentrantTemplate(0x03)); err != nil {
		t.Fatalf("set reentrant code for contract2: %v", err)
	}
	if err := st.SetCode("contract3", reentrantTemplate(0x04)); err != nil {
		t.Fatalf("set reentrant code for contract3: %v", err)
	}
	if err := st.SetCode("contract4", reentrantTemplate(0x05)); err != nil {
		t.Fatalf("set reentrant code for contract4: %v", err)
	}
	if err := st.SetCode("contract5", reentrantTemplate(0x06)); err != nil {
		t.Fatalf("set reentrant code for contract5: %v", err)
	}

	entryCode := []byte{
		OpPUSH1, 0x10,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpDELEGATECALL,

		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpMLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", entryCode); err != nil {
		t.Fatalf("set entry code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x01),
		word32FromUint64(0x01),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected depth-limit delegatecall data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_DELEGATECALL_FailureCanBeAbsorbedByCaller(t *testing.T) {
	st := vmstate.New()

	libraryCode := []byte{
		OpPUSH1, 0x2a,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpREVERT,
	}
	if err := st.SetCode("contract2", libraryCode); err != nil {
		t.Fatalf("set library code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x20,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpDELEGATECALL,

		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x00),
		word32FromUint64(0x2a),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected absorbed delegatecall data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_DELEGATECALL_FailureCanBePropagatedByCaller(t *testing.T) {
	st := vmstate.New()

	libraryCode := []byte{
		OpPUSH1, 0x7e,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpREVERT,
	}
	if err := st.SetCode("contract2", libraryCode); err != nil {
		t.Fatalf("set library code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpDELEGATECALL,

		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpRETURNDATACOPY,

		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpREVERT,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0x7e)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected propagated delegatecall data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_STATICCALL_AllowsReadOnlyExecution(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x44,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpCALLVALUE,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpADDRESS,
		OpPUSH1, 0x40,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x60,
		OpRETURN,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x60,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpSTATICCALL,

		OpPUSH1, 0x00,
		OpPUSH1, 0x60,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		Value:    0x09,
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x44),
		word32FromUint64(0x00),
		word32FromUint64(0x02),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected staticcall read payload: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_STATICCALL_RejectsSSTORE(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x55,
		OpPUSH1, 0x01,
		OpSSTORE,
		OpSTOP,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x08,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpSTATICCALL,
		OpPUSH1, 0x08,
		OpMSTORE,
		OpPUSH1, 0x20,
		OpMLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x10,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if !errors.Is(err, ErrWriteProtection) {
		t.Fatalf("expected ErrWriteProtection, got=%v", err)
	}

	if _, ok := st.GetStorage("contract2", "0x1"); ok {
		t.Fatalf("staticcall must not mutate storage")
	}
}

func TestExecutor_Execute_DELEGATECALL_InsideStaticContextRejectsSSTOREOnCallerStorage(t *testing.T) {
	st := vmstate.New()

	libraryCode := []byte{
		OpPUSH1, 0x77,
		OpPUSH1, 0x01,
		OpSSTORE,
		OpSTOP,
	}
	if err := st.SetCode("contract3", libraryCode); err != nil {
		t.Fatalf("set library code: %v", err)
	}

	staticCalleeCode := []byte{
		OpPUSH1, 0x08,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x03,
		OpPUSH1, 0xff,
		OpDELEGATECALL,
		OpPUSH1, 0x08,
		OpMSTORE,
		OpPUSH1, 0x20,
		OpMLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x10,
		OpRETURN,
	}
	if err := st.SetCode("contract2", staticCalleeCode); err != nil {
		t.Fatalf("set static callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x10,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpSTATICCALL,
		OpPUSH1, 0x20,
		OpPUSH1, 0x10,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if !errors.Is(err, ErrWriteProtection) {
		t.Fatalf("expected ErrWriteProtection, got=%v", err)
	}

	if _, ok := st.GetStorage("contract1", "0x1"); ok {
		t.Fatalf("static context must prevent delegatecall storage mutation too")
	}
}

func TestExecutor_Execute_STATICCALL_RETURNDATASIZE_RETURNDATACOPY_AfterSuccess(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x4d,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpSTATICCALL,

		OpRETURNDATASIZE,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURNDATACOPY,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x20),
		word32FromUint64(0x4d),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected staticcall returndata success payload: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_STATICCALL_RETURNDATASIZE_RETURNDATACOPY_AfterRevert(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpPUSH1, 0x7e,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpREVERT,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpSTATICCALL,

		OpRETURNDATASIZE,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURNDATACOPY,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x20),
		word32FromUint64(0x7e),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected staticcall returndata revert payload: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_STATICCALL_ToMissingAccountZerosReturnAreaAndSucceeds(t *testing.T) {
	st := vmstate.New()

	callerCode := []byte{
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x09,
		OpPUSH1, 0xff,
		OpSTATICCALL,

		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0x01)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected staticcall missing-account payload: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_STATICCALL_ToExistingNoCodeAccountZerosReturnAreaAndSucceeds(t *testing.T) {
	st := vmstate.New()
	if _, err := st.EnsureAccount("contract7"); err != nil {
		t.Fatalf("ensure account: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x07,
		OpPUSH1, 0xff,
		OpSTATICCALL,

		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0x01)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected staticcall no-code payload: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_STATICCALL_DepthLimitReturnsZeroAndZerosRetArea(t *testing.T) {
	st := vmstate.New()

	leafCode := []byte{
		OpPUSH1, 0xee,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract6", leafCode); err != nil {
		t.Fatalf("set leaf code: %v", err)
	}

	reentrantTemplate := func(target byte) []byte {
		return []byte{
			OpPUSH1, 0x20,
			OpPUSH1, 0x00,
			OpPUSH1, 0x00,
			OpPUSH1, 0x00,
			OpPUSH1, target,
			OpPUSH1, 0xff,
			OpSTATICCALL,

			OpPUSH1, 0x00,
			OpMSTORE,

			OpPUSH1, 0x00,
			OpPUSH1, 0x20,
			OpRETURN,
		}
	}

	if err := st.SetCode("contract2", reentrantTemplate(0x03)); err != nil {
		t.Fatalf("set reentrant code for contract2: %v", err)
	}
	if err := st.SetCode("contract3", reentrantTemplate(0x04)); err != nil {
		t.Fatalf("set reentrant code for contract3: %v", err)
	}
	if err := st.SetCode("contract4", reentrantTemplate(0x05)); err != nil {
		t.Fatalf("set reentrant code for contract4: %v", err)
	}
	if err := st.SetCode("contract5", reentrantTemplate(0x06)); err != nil {
		t.Fatalf("set reentrant code for contract5: %v", err)
	}

	entryCode := []byte{
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpPUSH1, 0xff,
		OpSTATICCALL,

		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", entryCode); err != nil {
		t.Fatalf("set entry code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0x01)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected staticcall depth-limit payload: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_SSTORE_ClearAddsRefund(t *testing.T) {
	st := vmstate.New()

	if err := st.SetStorage("contract1", storageKey(1), storageValue(7)); err != nil {
		t.Fatalf("set initial storage: %v", err)
	}

	code := []byte{
		OpPUSH1, 0x00,
		OpPUSH1, 0x01,
		OpSSTORE,
		OpSTOP,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	gotRaw, ok := st.GetStorage("contract1", storageKey(1))
	if !ok {
		t.Fatalf("expected cleared storage slot to remain addressable")
	}
	if parseStorageValue(gotRaw) != 0 {
		t.Fatalf("expected cleared storage slot value to be zero: got=%s", gotRaw)
	}

	fullGasUsed := uint64(21000 + len(code))
	expectedRefund := fullGasUsed / 5
	expectedGasUsed := fullGasUsed - expectedRefund

	if result.GasUsed != expectedGasUsed {
		t.Fatalf("unexpected gas used after clear refund: got=%d want=%d", result.GasUsed, expectedGasUsed)
	}
}
func TestExecutor_Execute_SSTORE_ZeroToZeroDoesNotAddRefund(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpPUSH1, 0x00,
		OpPUSH1, 0x01,
		OpSSTORE,
		OpSTOP,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	gotRaw, ok := st.GetStorage("contract1", storageKey(1))
	if !ok {
		t.Fatalf("expected zeroed storage slot to remain addressable")
	}
	if parseStorageValue(gotRaw) != 0 {
		t.Fatalf("expected zeroed storage slot value to remain zero: got=%s", gotRaw)
	}

	expectedGasUsed := uint64(21000 + len(code))
	if result.GasUsed != expectedGasUsed {
		t.Fatalf("unexpected gas used for zero-to-zero sstore: got=%d want=%d", result.GasUsed, expectedGasUsed)
	}
}
func TestExecutor_Execute_SSTORE_SameValueDoesNotAddRefund(t *testing.T) {
	st := vmstate.New()

	if err := st.SetStorage("contract1", storageKey(1), storageValue(7)); err != nil {
		t.Fatalf("set initial storage: %v", err)
	}

	code := []byte{
		OpPUSH1, 0x07,
		OpPUSH1, 0x01,
		OpSSTORE,
		OpSTOP,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	gotRaw, ok := st.GetStorage("contract1", storageKey(1))
	if !ok {
		t.Fatalf("expected storage slot to remain addressable")
	}
	if parseStorageValue(gotRaw) != 7 {
		t.Fatalf("expected storage slot to remain unchanged: got=%s", gotRaw)
	}

	expectedGasUsed := uint64(21000 + len(code))
	if result.GasUsed != expectedGasUsed {
		t.Fatalf("unexpected gas used for same-value sstore: got=%d want=%d", result.GasUsed, expectedGasUsed)
	}
}

func TestExecutor_Execute_SSTORE_NonZeroToNonZeroDoesNotAddRefund(t *testing.T) {
	st := vmstate.New()

	if err := st.SetStorage("contract1", storageKey(1), storageValue(7)); err != nil {
		t.Fatalf("set initial storage: %v", err)
	}

	code := []byte{
		OpPUSH1, 0x09,
		OpPUSH1, 0x01,
		OpSSTORE,
		OpSTOP,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	gotRaw, ok := st.GetStorage("contract1", storageKey(1))
	if !ok {
		t.Fatalf("expected storage slot to remain addressable")
	}
	if parseStorageValue(gotRaw) != 9 {
		t.Fatalf("expected storage slot to update to non-zero value: got=%s", gotRaw)
	}

	expectedGasUsed := uint64(21000 + len(code))
	if result.GasUsed != expectedGasUsed {
		t.Fatalf("unexpected gas used for nonzero-to-nonzero sstore: got=%d want=%d", result.GasUsed, expectedGasUsed)
	}
}
func TestForwardedCallGas_WithValueAddsStipendAfterCap(t *testing.T) {
	got := forwardedCallGas(1000, 1000, 1)
	want := uint64(985 + 2300)

	if got != want {
		t.Fatalf("unexpected forwarded call gas with stipend: got=%d want=%d", got, want)
	}
}
func TestForwardedGas_CapsToSixtyThreeSixtyFourthsOfAvailable(t *testing.T) {
	got := forwardedGas(1000, 1000)
	want := uint64(1000 - (1000 / 64))

	if got != want {
		t.Fatalf("unexpected forwarded gas cap: got=%d want=%d", got, want)
	}
}
func TestDeriveCreate2Address_SameInputsAreDeterministic(t *testing.T) {
	initCode := []byte{OpPUSH1, 0x01, OpPUSH1, 0x00, OpRETURN}

	first := deriveCreate2Address("alice", 7, initCode)
	second := deriveCreate2Address("alice", 7, initCode)

	if first != second {
		t.Fatalf("expected deterministic create2 address: first=%s second=%s", first, second)
	}
}

func TestDeriveCreate2Address_DifferentSaltChangesAddress(t *testing.T) {
	initCode := []byte{OpPUSH1, 0x01, OpPUSH1, 0x00, OpRETURN}

	first := deriveCreate2Address("alice", 7, initCode)
	second := deriveCreate2Address("alice", 8, initCode)

	if first == second {
		t.Fatalf("expected different create2 addresses for different salts: first=%s second=%s", first, second)
	}
}

func TestDeriveCreate2Address_DifferentInitCodeChangesAddress(t *testing.T) {
	firstCode := []byte{OpPUSH1, 0x01, OpPUSH1, 0x00, OpRETURN}
	secondCode := []byte{OpPUSH1, 0x02, OpPUSH1, 0x00, OpRETURN}

	first := deriveCreate2Address("alice", 7, firstCode)
	second := deriveCreate2Address("alice", 7, secondCode)

	if first == second {
		t.Fatalf("expected different create2 addresses for different init code: first=%s second=%s", first, second)
	}
}
func TestDeriveCreateAddress_DifferentEOACreatorsProduceDistinctFirstAddresses(t *testing.T) {
	alice := deriveCreateAddress("alice", 1)
	bob := deriveCreateAddress("bob", 1)

	if alice == bob {
		t.Fatalf("expected distinct first create addresses for different creators: alice=%s bob=%s", alice, bob)
	}
}
func TestExecutor_Execute_CREATE2_DifferentSaltProducesDifferentAddress(t *testing.T) {
	st := vmstate.New()

	initCode := []byte{
		OpPUSH1, 0x65,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}

	firstCallerCode := append([]byte{
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x18,
		OpPUSH1, 0x00,
		OpCODECOPY,

		OpPUSH1, 0x07,
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpCREATE2,

		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}, initCode...)

	secondCallerCode := append([]byte{
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x18,
		OpPUSH1, 0x00,
		OpCODECOPY,

		OpPUSH1, 0x08,
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpCREATE2,

		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}, initCode...)

	if err := st.SetCode("contract1", firstCallerCode); err != nil {
		t.Fatalf("set first caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	firstResult, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute first create2: %v", err)
	}

	if err := st.SetCode("contract1", secondCallerCode); err != nil {
		t.Fatalf("set second caller code: %v", err)
	}

	secondResult, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute second create2: %v", err)
	}

	firstWord := trailingUint64(firstResult.ReturnData)
	secondWord := trailingUint64(secondResult.ReturnData)

	if firstWord == 0 || secondWord == 0 {
		t.Fatalf("expected non-zero create2 return words: first=%d second=%d", firstWord, secondWord)
	}
	if firstWord == secondWord {
		t.Fatalf("expected different create2 addresses for different salts: first=%d second=%d", firstWord, secondWord)
	}
}
func TestExecutor_Execute_CREATE2_PassesValueIntoInitCode(t *testing.T) {
	st := vmstate.New()

	initCode := []byte{
		OpCALLVALUE,
		OpPUSH1, 0x00,
		OpSSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpRETURN,
	}

	callerCode := append([]byte{
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x18,
		OpPUSH1, 0x00,
		OpCODECOPY,

		OpPUSH1, 0x07,
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x00,
		OpPUSH1, 0x09,
		OpCREATE2,

		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}, initCode...)

	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	created := contractAddressFromWord(trailingUint64(result.ReturnData))
	if created == "contract0" {
		t.Fatalf("expected non-zero created contract address")
	}

	got, ok := st.GetStorage(created, storageKey(0))
	if !ok {
		t.Fatalf("expected storage slot 0 to be set")
	}
	if got != storageValue(0x09) {
		t.Fatalf("unexpected stored value: got=%s want=%s", got, storageValue(0x09))
	}
}
func TestExecutor_Execute_CREATE2_ReturnsDeterministicAddressAndCreatesCallableAccount(t *testing.T) {
	st := vmstate.New()

	initCode := []byte{
		OpPUSH1, 0x65,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}

	callerCode := append([]byte{
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x18,
		OpPUSH1, 0x00,
		OpCODECOPY,

		OpPUSH1, 0x07,
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpCREATE2,

		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}, initCode...)

	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(result.ReturnData) != 32 {
		t.Fatalf("unexpected create2 payload length: got=%d want=32", len(result.ReturnData))
	}
	if trailingUint64(result.ReturnData) == 0 {
		t.Fatalf("expected non-zero created contract word in return data")
	}

	created := contractAddressFromWord(trailingUint64(result.ReturnData))
	acct, ok := st.GetAccount(created)
	if !ok {
		t.Fatalf("expected created account %s to exist", created)
	}
	if len(acct.Code) == 0 {
		t.Fatalf("expected created account %s to have code", created)
	}
}
func TestExecutor_Execute_CREATE_ReturnsDeterministicAddressAndCreatesCallableAccount(t *testing.T) {
	st := vmstate.New()

	initCode := []byte{
		OpPUSH1, 0x65,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}

	callerCode := append([]byte{
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpCREATE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}, initCode...)

	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(result.ReturnData) != 32 {
		t.Fatalf("unexpected create payload length: got=%d want=32", len(result.ReturnData))
	}
	if trailingUint64(result.ReturnData) == 0 {
		t.Fatalf("expected non-zero created contract word in return data")
	}
}

func TestExecutor_Execute_CREATE_RevertReturnsZeroAndCopiesRevertData(t *testing.T) {
	st := vmstate.New()

	initCode := []byte{
		OpPUSH1, 0x7e,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpREVERT,
	}

	callerCode := append([]byte{
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x21,
		OpPUSH1, 0x00,
		OpCODECOPY,

		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpCREATE,

		OpPUSH1, 0x20,
		OpMSTORE,

		OpRETURNDATASIZE,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURNDATACOPY,

		OpPUSH1, 0x00,
		OpPUSH1, 0x60,
		OpRETURN,
	}, initCode...)

	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(0x20),
		word32FromUint64(0x7e),
		word32FromUint64(0x00),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected create revert payload: got=%v want=%v", result.ReturnData, want)
	}
}
func TestExecutor_Execute_CODESIZE_ReturnsCurrentCodeLength(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpCODESIZE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(uint64(len(code)))
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected codesize result: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CODECOPY_CopiesCodeBytesIntoMemory(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpPUSH1, 0x04, // size
		OpPUSH1, 0x00, // code offset
		OpPUSH1, 0x20, // mem offset
		OpCODECOPY,
		OpPUSH1, 0x20,
		OpPUSH1, 0x04,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := []byte{
		OpPUSH1, 0x04, OpPUSH1, 0x00,
	}
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected codecopy result: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CODECOPY_ZeroPadsOutOfBounds(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpPUSH1, 0x0a, // size
		OpPUSH1, 0x08, // code offset near end
		OpPUSH1, 0x20, // mem offset
		OpCODECOPY,
		OpPUSH1, 0x20,
		OpPUSH1, 0x06,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := []byte{
		0x20, OpPUSH1, 0x06, OpRETURN, 0x00, 0x00,
	}
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected zero-padded codecopy result: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CREATE_PassesValueIntoInitCode(t *testing.T) {
	st := vmstate.New()

	initCode := []byte{
		OpCALLVALUE,
		OpPUSH1, 0x00,
		OpSSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpRETURN,
	}

	callerCode := append([]byte{
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x16,
		OpPUSH1, 0x00,
		OpCODECOPY,

		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x00,
		OpPUSH1, 0x09,
		OpCREATE,

		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}, initCode...)

	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	created := contractAddressFromWord(trailingUint64(result.ReturnData))
	if created == "contract0" {
		t.Fatalf("expected non-zero created contract address")
	}

	got, ok := st.GetStorage(created, storageKey(0))
	if !ok {
		t.Fatalf("expected storage slot 0 to be set")
	}
	if got != storageValue(0x09) {
		t.Fatalf("unexpected stored value: got=%s want=%s", got, storageValue(0x09))
	}
}
func TestExecutor_Execute_CREATE_InstallsRuntimeCodeFromInitCodeUsingCodeCopy(t *testing.T) {
	st := vmstate.New()

	initCode := []byte{
		OpPUSH1, 0x20,
		OpPUSH1, 0x0c,
		OpPUSH1, 0x00,
		OpCODECOPY,
		OpPUSH1, 0x20,
		OpPUSH1, 0x00,
		OpRETURN,

		OpPUSH1, 0x77,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}

	callerCode := append([]byte{
		OpPUSH1, byte(len(initCode)),
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpCREATE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}, initCode...)

	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(result.ReturnData) != 32 {
		t.Fatalf("unexpected create payload length: got=%d want=32", len(result.ReturnData))
	}
	if trailingUint64(result.ReturnData) == 0 {
		t.Fatalf("expected non-zero created contract word in return data")
	}
}

func TestExecutor_Execute_MSTORE8_WritesLowestByteOnly(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPUSH1, 0xab,
		OpPUSH1, 0x00,
		OpMSTORE8,
		OpPUSH1, 0x00,
		OpMLOAD,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x08,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := []byte{0xab, 0, 0, 0, 0, 0, 0, 0}
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected mstore8 return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_MSTORE8_WritesAtExactOffset(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPUSH1, 0x11,
		OpPUSH1, 0x00,
		OpMSTORE8,
		OpPUSH1, 0x22,
		OpPUSH1, 0x01,
		OpMSTORE8,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := []byte{0x11, 0x22}
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected exact-offset mstore8 return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_MSIZE_ZeroBeforeMemoryUse(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpMSIZE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected msize before memory use: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_MSIZE_AfterMSTORE_IsThirtyTwo(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPUSH1, 0x2a,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpMSIZE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(32)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected msize after mstore: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_MSIZE_AfterMSTORE8_TracksExactBytes(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPUSH1, 0x11,
		OpPUSH1, 0x01,
		OpMSTORE8,
		OpMSIZE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(2)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected msize after mstore8: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_PC_AtEntryIsZero(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPC,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected pc at entry: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_PC_AfterPushReflectsOpcodePosition(t *testing.T) {
	st := vmstate.New()
	code := []byte{
		OpPUSH1, 0xaa,
		OpPC,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(2)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected pc after push: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_EXTCODESIZE_ReturnsTargetCodeSize(t *testing.T) {
	st := vmstate.New()

	targetCode := []byte{
		OpPUSH1, 0x2a,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x08,
		OpRETURN,
	}
	if err := st.SetCode("contract2", targetCode); err != nil {
		t.Fatalf("set target code: %v", err)
	}

	code := []byte{
		OpPUSH1, 0x02,
		OpEXTCODESIZE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(uint64(len(targetCode)))
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected extcodesize return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_EXTCODESIZE_NoCodeAccountReturnsZero(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpPUSH1, 0x09,
		OpEXTCODESIZE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected extcodesize no-code return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_EXTCODECOPY_CopiesTargetCode(t *testing.T) {
	st := vmstate.New()

	targetCode := []byte{
		OpPUSH1, 0x2a,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x08,
		OpRETURN,
	}
	if err := st.SetCode("contract2", targetCode); err != nil {
		t.Fatalf("set target code: %v", err)
	}

	code := []byte{
		OpPUSH1, 0x0a, // size
		OpPUSH1, 0x00, // code offset
		OpPUSH1, 0x00, // mem offset
		OpPUSH1, 0x02, // address word -> contract2
		OpEXTCODECOPY,
		OpPUSH1, 0x00,
		OpPUSH1, 0x0a,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := targetCode
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected extcodecopy return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_EXTCODECOPY_OutOfBoundsZeroPads(t *testing.T) {
	st := vmstate.New()

	targetCode := []byte{OpPUSH1, 0x2a}
	if err := st.SetCode("contract2", targetCode); err != nil {
		t.Fatalf("set target code: %v", err)
	}

	code := []byte{
		OpPUSH1, 0x04, // size
		OpPUSH1, 0x00, // code offset
		OpPUSH1, 0x00, // mem offset
		OpPUSH1, 0x02, // address word -> contract2
		OpEXTCODECOPY,
		OpPUSH1, 0x00,
		OpPUSH1, 0x04,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := []byte{OpPUSH1, 0x2a, 0x00, 0x00}
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected extcodecopy zero-pad return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_SELFDESTRUCT_TransfersBalanceAndClearsAccount(t *testing.T) {
	st := vmstate.New()

	if err := st.SetBalance("contract1", 55); err != nil {
		t.Fatalf("set contract balance: %v", err)
	}
	if err := st.SetStorage("contract1", storageKey(1), storageValue(9)); err != nil {
		t.Fatalf("set contract storage: %v", err)
	}

	code := []byte{
		OpPUSH1, 0x02,
		OpSELFDESTRUCT,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected selfdestruct execution success")
	}

	beneficiary, ok := st.GetAccount("contract2")
	if !ok {
		t.Fatalf("expected beneficiary account to exist")
	}
	if beneficiary.Balance != 55 {
		t.Fatalf("unexpected beneficiary balance: got=%d want=55", beneficiary.Balance)
	}

	destroyed, ok := st.GetAccount("contract1")
	if !ok {
		t.Fatalf("expected destroyed account shell to remain addressable")
	}
	if destroyed.Balance != 0 {
		t.Fatalf("expected destroyed account balance to be zero: got=%d", destroyed.Balance)
	}
	if len(destroyed.Code) != 0 {
		t.Fatalf("expected destroyed account code to be cleared")
	}
	if len(destroyed.Storage) != 0 {
		t.Fatalf("expected destroyed account storage to be cleared")
	}
}
func TestExecutor_Execute_SLOAD_RepeatedAccessUsesWarmThenColdAccounting(t *testing.T) {
	st := vmstate.New()

	if err := st.SetStorage("contract1", storageKey(1), storageValue(44)); err != nil {
		t.Fatalf("set storage: %v", err)
	}

	code := []byte{
		OpPUSH1, 0x01,
		OpSLOAD,
		OpPUSH1, 0x01,
		OpSLOAD,
		OpSTOP,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 2199,
	})
	if err == nil {
		t.Fatalf("expected out of gas for cold+warm sload below 2200 gas")
	}

	_, err = exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 2200,
	})
	if err != nil {
		t.Fatalf("expected success for cold+warm sload at 2200 gas, got=%v", err)
	}
}
func TestExecutor_Execute_BALANCE_RepeatedAccessUsesColdThenWarmAccounting(t *testing.T) {
	st := vmstate.New()

	if err := st.SetBalance("contract2", 77); err != nil {
		t.Fatalf("set target balance: %v", err)
	}

	code := []byte{
		OpPUSH1, 0x02,
		OpBALANCE,
		OpPUSH1, 0x02,
		OpBALANCE,
		OpSTOP,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 2699,
	})
	if err == nil {
		t.Fatalf("expected out of gas for cold+warm balance access below 2700 gas")
	}

	_, err = exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 2700,
	})
	if err != nil {
		t.Fatalf("expected success for cold+warm balance access at 2700 gas, got=%v", err)
	}
}
func TestExecutor_Execute_EXTCODEHASH_ReturnsNonZeroForCodeAccountAndZeroForNoCode(t *testing.T) {
	st := vmstate.New()

	targetCode := []byte{OpPUSH1, 0x2a, OpPUSH1, 0x00, OpMSTORE, OpPUSH1, 0x00, OpPUSH1, 0x20, OpRETURN}
	if err := st.SetCode("contract2", targetCode); err != nil {
		t.Fatalf("set target code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x02,
		OpEXTCODEHASH,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x03,
		OpEXTCODEHASH,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	if _, err := st.EnsureAccount("contract3"); err != nil {
		t.Fatalf("ensure no-code account: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := concatBytes(
		word32FromUint64(uint64(len(targetCode))),
		word32FromUint64(0),
	)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected extcodehash return data: got=%v want=%v", result.ReturnData, want)
	}
}
func TestExecutor_Execute_EXTCODECOPY_NoCodeAccountReturnsZeroBytes(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpPUSH1, 0x03, // size
		OpPUSH1, 0x00, // code offset
		OpPUSH1, 0x00, // mem offset
		OpPUSH1, 0x09, // missing account
		OpEXTCODECOPY,
		OpPUSH1, 0x00,
		OpPUSH1, 0x03,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := []byte{0x00, 0x00, 0x00}
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected extcodecopy no-code return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CALLDATACOPY_CopiesInputToMemory(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpPUSH1, 0x04, // size
		OpPUSH1, 0x00, // data offset
		OpPUSH1, 0x00, // mem offset
		OpCALLDATACOPY,
		OpPUSH1, 0x00,
		OpPUSH1, 0x04,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
		Data:     []byte{0xaa, 0xbb, 0xcc, 0xdd},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := []byte{0xaa, 0xbb, 0xcc, 0xdd}
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected calldatacopy return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CALLDATACOPY_OutOfBoundsZeroPads(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpPUSH1, 0x04, // size
		OpPUSH1, 0x02, // data offset
		OpPUSH1, 0x00, // mem offset
		OpCALLDATACOPY,
		OpPUSH1, 0x00,
		OpPUSH1, 0x04,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
		Data:     []byte{0xaa, 0xbb},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := []byte{0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected calldatacopy zero-pad return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CALLDATACOPY_WithOffsetCopiesSuffix(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpPUSH1, 0x02, // size
		OpPUSH1, 0x02, // data offset
		OpPUSH1, 0x00, // mem offset
		OpCALLDATACOPY,
		OpPUSH1, 0x00,
		OpPUSH1, 0x02,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
		Data:     []byte{0x11, 0x22, 0x33, 0x44},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := []byte{0x33, 0x44}
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected calldatacopy suffix return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_GAS_PushesNonZeroValue(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpGAS,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(result.ReturnData) != 32 {
		t.Fatalf("unexpected return data length: got=%d want=32", len(result.ReturnData))
	}

	if trailingUint64(result.ReturnData) == 0 {
		t.Fatalf("expected GAS to push non-zero value")
	}
}

func TestExecutor_Execute_GAS_AfterPushIsStableOrLowerButNonZero(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpGAS,
		OpPUSH1, 0x00,
		OpMSTORE,

		OpPUSH1, 0x2a,

		OpGAS,
		OpPUSH1, 0x20,
		OpMSTORE,

		OpPUSH1, 0x00,
		OpPUSH1, 0x40,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(result.ReturnData) != 64 {
		t.Fatalf("unexpected return data length: got=%d want=64", len(result.ReturnData))
	}

	first := result.ReturnData[:32]
	second := result.ReturnData[32:64]

	if trailingUint64(first) == 0 {
		t.Fatalf("expected first GAS value to be non-zero")
	}
	if trailingUint64(second) == 0 {
		t.Fatalf("expected second GAS value to be non-zero")
	}
}

func TestExecutor_Execute_BALANCE_ReturnsTargetBalance(t *testing.T) {
	st := vmstate.New()

	if err := st.SetBalance("contract2", 1234); err != nil {
		t.Fatalf("set balance: %v", err)
	}

	code := []byte{
		OpPUSH1, 0x02,
		OpBALANCE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(1234)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected balance return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_BALANCE_MissingAccountReturnsZero(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpPUSH1, 0x09,
		OpBALANCE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected missing-account balance return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_SELFBALANCE_ReturnsContractBalance(t *testing.T) {
	st := vmstate.New()

	if err := st.SetBalance("contract1", 4321); err != nil {
		t.Fatalf("set balance: %v", err)
	}

	code := []byte{
		OpSELFBALANCE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(4321)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected selfbalance return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_SELFBALANCE_ZeroWhenContractBalanceUnset(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpSELFBALANCE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected zero selfbalance return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_ORIGIN_ReturnsTopLevelSender(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpORIGIN,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected origin return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_ORIGIN_PreservedAcrossCALL(t *testing.T) {
	st := vmstate.New()

	calleeCode := []byte{
		OpORIGIN,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract2", calleeCode); err != nil {
		t.Fatalf("set callee code: %v", err)
	}

	callerCode := []byte{
		OpPUSH1, 0x20, // out size
		OpPUSH1, 0x00, // out offset
		OpPUSH1, 0x00, // in size
		OpPUSH1, 0x00, // in offset
		OpPUSH1, 0x00, // value
		OpPUSH1, 0x02, // to
		OpPUSH1, 0xff, // gas
		OpCALL,
		OpPUSH1, 0x00,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURNDATACOPY,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", callerCode); err != nil {
		t.Fatalf("set caller code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected preserved origin return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_NUMBER_ReturnsExecutionContextBlockNumber(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpNUMBER,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{
		BlockNumber: 77,
	}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(77)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected number return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_NUMBER_ZeroWhenExecutionContextUnset(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpNUMBER,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected zero number return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_TIMESTAMP_ReturnsExecutionContextTimestamp(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpTIMESTAMP,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{
		Timestamp: 123456789,
	}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(123456789)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected timestamp return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_TIMESTAMP_ZeroWhenExecutionContextUnset(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpTIMESTAMP,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected zero timestamp return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_BASEFEE_ReturnsExecutionContextBaseFee(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpBASEFEE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{
		BaseFee: 25,
	}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(25)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected basefee return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_BASEFEE_ZeroWhenExecutionContextUnset(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpBASEFEE,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected zero basefee return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_GASLIMIT_ReturnsExecutionContextGasLimit(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpGASLIMIT,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{
		GasLimit: 5000000,
	}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(5000000)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected gaslimit return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_GASLIMIT_ZeroWhenExecutionContextUnset(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpGASLIMIT,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected zero gaslimit return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CHAINID_ReturnsExecutionContextChainID(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpCHAINID,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{
		ChainID: 1001,
	}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(1001)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected chainid return data: got=%v want=%v", result.ReturnData, want)
	}
}

func TestExecutor_Execute_CHAINID_ZeroWhenExecutionContextUnset(t *testing.T) {
	st := vmstate.New()

	code := []byte{
		OpCHAINID,
		OpPUSH1, 0x00,
		OpMSTORE,
		OpPUSH1, 0x00,
		OpPUSH1, 0x20,
		OpRETURN,
	}
	if err := st.SetCode("contract1", code); err != nil {
		t.Fatalf("set code: %v", err)
	}

	exec, err := New(st)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Execute(ExecutionContext{}, Message{
		From:     "alice",
		To:       "contract1",
		GasLimit: 50000,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := word32FromUint64(0)
	if !bytes.Equal(result.ReturnData, want) {
		t.Fatalf("unexpected zero chainid return data: got=%v want=%v", result.ReturnData, want)
	}
}
