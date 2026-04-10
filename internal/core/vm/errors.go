package vm

import "errors"

var (
	ErrStackUnderflow      = errors.New("vm: stack underflow")
	ErrStackOverflow       = errors.New("vm: stack overflow")
	ErrInvalidOpcode       = errors.New("vm: invalid opcode")
	ErrInvalidJump         = errors.New("vm: invalid jump")
	ErrOutOfGas            = errors.New("vm: out of gas")
	ErrWriteProtection     = errors.New("vm: write protection in static context")
	ErrDepthLimitExceeded  = errors.New("vm: call depth limit exceeded")
	ErrCodeSizeLimit       = errors.New("vm: code size limit exceeded")
	ErrInitCodeSizeLimit   = errors.New("vm: init code size limit exceeded")
	ErrMemoryLimitExceeded = errors.New("vm: memory limit exceeded")
	ErrExecutionAborted    = errors.New("vm: execution aborted")
)
