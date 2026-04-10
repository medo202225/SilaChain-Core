package vm

import "math/big"

type Interpreter struct {
	limits Limits
	host   Host
}

func NewInterpreter(limits Limits) *Interpreter {
	return &Interpreter{limits: limits}
}

func NewInterpreterWithHost(limits Limits, host Host) *Interpreter {
	return &Interpreter{
		limits: limits,
		host:   host,
	}
}

func addressToWord(addr string) *big.Int {
	return NewWordFromBytes([]byte(addr))
}

func calldataLoad(input []byte, offset uint64) *big.Int {
	buf := make([]byte, 32)
	if offset >= uint64(len(input)) {
		return NewWordFromBytes(buf)
	}

	end := offset + 32
	if end > uint64(len(input)) {
		end = uint64(len(input))
	}

	copy(buf, input[offset:end])
	return NewWordFromBytes(buf)
}

func isTruthy(v *big.Int) bool {
	return v != nil && v.Sign() != 0
}

func boolWord(ok bool) *big.Int {
	if ok {
		return NewWordFromUint64(1)
	}
	return NewWordFromUint64(0)
}

func (in *Interpreter) isValidJumpDest(frame *CallFrame, dest uint64) bool {
	if dest >= uint64(len(frame.Code)) {
		return false
	}
	return frame.Code[dest] == OpJumpDest
}

func (in *Interpreter) runLog(frame *CallFrame, topicCount int) error {
	if frame.Context.Static {
		return ErrWriteProtection
	}
	if in.host == nil {
		return ErrExecutionAborted
	}

	offsetWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	sizeWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}

	offset := offsetWord.Uint64()
	size := sizeWord.Uint64()

	data, err := frame.Memory.Load(offset, size)
	if err != nil {
		return err
	}

	topics := make([]string, 0, topicCount)
	for i := 0; i < topicCount; i++ {
		topicWord, err := frame.Stack.Pop()
		if err != nil {
			return err
		}
		topics = append(topics, string(WordToBytes32(topicWord)))
	}

	in.host.EmitLog(LogEntry{
		Address: frame.Context.ContractAddr,
		Topics:  topics,
		Data:    data,
	})

	return nil
}

func (in *Interpreter) runCallCommon(frame *CallFrame, forceStatic bool) error {
	if in.host == nil {
		return ErrExecutionAborted
	}

	gasWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	addrWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	valueWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	inOffsetWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	inSizeWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	outOffsetWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	outSizeWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}

	callGas := gasWord.Uint64()
	target := string(WordToBytes32(addrWord))
	value := valueWord.Uint64()
	inOffset := inOffsetWord.Uint64()
	inSize := inSizeWord.Uint64()
	outOffset := outOffsetWord.Uint64()
	outSize := outSizeWord.Uint64()

	if forceStatic && value != 0 {
		if err := frame.Stack.Push(NewWordFromUint64(0)); err != nil {
			return err
		}
		return nil
	}

	input, err := frame.Memory.Load(inOffset, inSize)
	if err != nil {
		return err
	}

	result := in.host.CallContract(
		frame.Context.ContractAddr,
		target,
		input,
		value,
		callGas,
		frame.Context.Static || forceStatic,
	)

	if result.Err != nil || !result.Success {
		if err := frame.Stack.Push(NewWordFromUint64(0)); err != nil {
			return err
		}
		return nil
	}

	copySize := outSize
	if uint64(len(result.ReturnData)) < copySize {
		copySize = uint64(len(result.ReturnData))
	}

	if copySize > 0 {
		if err := frame.Memory.Store(outOffset, result.ReturnData[:copySize]); err != nil {
			return err
		}
	}

	if err := frame.Stack.Push(NewWordFromUint64(1)); err != nil {
		return err
	}
	return nil
}

func (in *Interpreter) runCall(frame *CallFrame) error {
	return in.runCallCommon(frame, false)
}

func (in *Interpreter) runStaticCall(frame *CallFrame) error {
	return in.runCallCommon(frame, true)
}

func (in *Interpreter) runCreate(frame *CallFrame) error {
	if frame.Context.Static {
		return ErrWriteProtection
	}
	if in.host == nil {
		return ErrExecutionAborted
	}

	valueWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	offsetWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	sizeWord, err := frame.Stack.Pop()
	if err != nil {
		return err
	}

	value := valueWord.Uint64()
	offset := offsetWord.Uint64()
	size := sizeWord.Uint64()

	initCode, err := frame.Memory.Load(offset, size)
	if err != nil {
		return err
	}

	newAddr, err := in.host.CreateContractAddress(frame.Context.ContractAddr)
	if err != nil {
		if err := frame.Stack.Push(NewWordFromUint64(0)); err != nil {
			return err
		}
		return nil
	}

	child := NewInterpreterWithHost(in.limits, in.host)
	childCtx := ExecutionContext{
		VMVersion:    frame.Context.VMVersion,
		ContractAddr: newAddr,
		CodeAddr:     newAddr,
		StorageAddr:  newAddr,
		Caller:       frame.Context.ContractAddr,
		Origin:       frame.Context.Origin,
		CallValue:    value,
		Input:        nil,
		GasRemaining: frame.Context.GasRemaining,
		Depth:        frame.Context.Depth + 1,
		Static:       false,
		Block:        frame.Context.Block,
		Tx:           frame.Context.Tx,
	}

	result := child.Run(childCtx, initCode)
	if !result.Succeeded() {
		_ = in.host.DeleteCode(newAddr)
		if err := frame.Stack.Push(NewWordFromUint64(0)); err != nil {
			return err
		}
		return nil
	}

	if err := in.host.SetCode(newAddr, result.ReturnData); err != nil {
		_ = in.host.DeleteCode(newAddr)
		if err := frame.Stack.Push(NewWordFromUint64(0)); err != nil {
			return err
		}
		return nil
	}

	frame.SetCreatedAddress(newAddr)

	if err := frame.Stack.Push(addressToWord(newAddr)); err != nil {
		return err
	}
	return nil
}

func (in *Interpreter) Run(ctx ExecutionContext, code []byte) ExecutionResult {
	if uint64(len(code)) > in.limits.MaxCodeSize {
		return FaultResult(ErrCodeSizeLimit, 0, ctx.GasRemaining, "")
	}

	frame := NewCallFrame(ctx, code, in.limits)

	checkpointID := -1
	if in.host != nil && !ctx.Static {
		checkpointID = in.host.CreateCheckpoint()
	}

	commitCheckpoint := func() error {
		if in.host != nil && checkpointID >= 0 {
			return in.host.CommitCheckpoint(checkpointID)
		}
		return nil
	}

	revertCheckpoint := func() error {
		if in.host != nil && checkpointID >= 0 {
			return in.host.RevertCheckpoint(checkpointID)
		}
		return nil
	}

	for {
		if frame.PC >= uint64(len(frame.Code)) {
			if err := commitCheckpoint(); err != nil {
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			gasRemaining := frame.RemainingGas()
			gasUsed := ctx.GasRemaining - gasRemaining
			return SuccessResult(nil, gasUsed, gasRemaining, nil, frame.LastCreatedAddress)
		}

		op := frame.Code[frame.PC]
		frame.PC++

		if err := frame.ConsumeGas(GasCost(op)); err != nil {
			_ = revertCheckpoint()
			return FaultResult(err, ctx.GasRemaining, frame.RemainingGas(), frame.LastCreatedAddress)
		}

		switch op {
		case OpStop:
			if err := commitCheckpoint(); err != nil {
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			gasRemaining := frame.RemainingGas()
			gasUsed := ctx.GasRemaining - gasRemaining
			return SuccessResult(nil, gasUsed, gasRemaining, nil, frame.LastCreatedAddress)

		case OpPop:
			if _, err := frame.Stack.Pop(); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpAdd:
			a, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			b, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			out := new(big.Int).Add(a, b)
			out = NormalizeWord(out)
			if err := frame.Stack.Push(out); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpSub:
			a, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			b, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			out := new(big.Int).Sub(a, b)
			out = NormalizeWord(out)
			if err := frame.Stack.Push(out); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpLT:
			a, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			b, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			if err := frame.Stack.Push(boolWord(b.Cmp(a) < 0)); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpGT:
			a, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			b, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			if err := frame.Stack.Push(boolWord(b.Cmp(a) > 0)); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpEQ:
			a, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			b, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			if err := frame.Stack.Push(boolWord(b.Cmp(a) == 0)); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpIsZero:
			v, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			if err := frame.Stack.Push(boolWord(!isTruthy(v))); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpAddress:
			if err := frame.Stack.Push(addressToWord(frame.Context.ContractAddr)); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpCaller:
			if err := frame.Stack.Push(addressToWord(frame.Context.Caller)); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpCallValue:
			if err := frame.Stack.Push(NewWordFromUint64(frame.Context.CallValue)); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpCallDataSize:
			if err := frame.Stack.Push(NewWordFromUint64(uint64(len(frame.Context.Input)))); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpCallDataLoad:
			offsetWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			value := calldataLoad(frame.Context.Input, offsetWord.Uint64())
			if err := frame.Stack.Push(value); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpMStore:
			offsetWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			valueWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

			offset := offsetWord.Uint64()
			value := WordToBytes32(valueWord)

			if err := frame.Memory.Store(offset, value); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpMLoad:
			offsetWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

			offset := offsetWord.Uint64()
			data, err := frame.Memory.Load(offset, 32)
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

			value := NewWordFromBytes(data)
			if err := frame.Stack.Push(value); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpSLoad:
			if in.host == nil {
				_ = revertCheckpoint()
				return FaultResult(ErrExecutionAborted, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

			keyWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

			key := string(WordToBytes32(keyWord))
			value := in.host.GetStorage(frame.Context.StorageAddr, key)

			if err := frame.Stack.Push(NewWordFromBytes(value)); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpSStore:
			if frame.Context.Static {
				_ = revertCheckpoint()
				return FaultResult(ErrWriteProtection, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			if in.host == nil {
				_ = revertCheckpoint()
				return FaultResult(ErrExecutionAborted, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

			keyWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			valueWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

			key := string(WordToBytes32(keyWord))
			value := WordToBytes32(valueWord)

			if err := in.host.SetStorage(frame.Context.StorageAddr, key, value); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpJump:
			destWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			dest := destWord.Uint64()
			if !in.isValidJumpDest(frame, dest) {
				_ = revertCheckpoint()
				return FaultResult(ErrInvalidJump, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			frame.PC = dest

		case OpJumpI:
			destWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			condWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			if isTruthy(condWord) {
				dest := destWord.Uint64()
				if !in.isValidJumpDest(frame, dest) {
					_ = revertCheckpoint()
					return FaultResult(ErrInvalidJump, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
				}
				frame.PC = dest
			}

		case OpJumpDest:

		case OpLog0:
			if err := in.runLog(frame, 0); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpLog1:
			if err := in.runLog(frame, 1); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpLog2:
			if err := in.runLog(frame, 2); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpLog3:
			if err := in.runLog(frame, 3); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpLog4:
			if err := in.runLog(frame, 4); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpCreate:
			if err := in.runCreate(frame); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpCall:
			if err := in.runCall(frame); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpStaticCall:
			if err := in.runStaticCall(frame); err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

		case OpReturn:
			offsetWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			sizeWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

			offset := offsetWord.Uint64()
			size := sizeWord.Uint64()

			data, err := frame.Memory.Load(offset, size)
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

			frame.SetReturnData(data)
			if err := commitCheckpoint(); err != nil {
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			gasRemaining := frame.RemainingGas()
			gasUsed := ctx.GasRemaining - gasRemaining
			return SuccessResult(frame.ReturnData, gasUsed, gasRemaining, nil, frame.LastCreatedAddress)

		case OpRevert:
			offsetWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}
			sizeWord, err := frame.Stack.Pop()
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

			offset := offsetWord.Uint64()
			size := sizeWord.Uint64()

			data, err := frame.Memory.Load(offset, size)
			if err != nil {
				_ = revertCheckpoint()
				return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
			}

			frame.SetRevertData(data)
			_ = revertCheckpoint()
			gasRemaining := frame.RemainingGas()
			gasUsed := ctx.GasRemaining - gasRemaining
			return RevertResult(frame.RevertData, gasUsed, gasRemaining, frame.LastCreatedAddress)

		default:
			if IsPush(op) {
				size := PushSize(op)
				if frame.PC+size > uint64(len(frame.Code)) {
					_ = revertCheckpoint()
					return FaultResult(ErrInvalidOpcode, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
				}

				data := frame.Code[frame.PC : frame.PC+size]
				frame.PC += size

				word := NewWordFromBytes(data)
				if err := frame.Stack.Push(word); err != nil {
					_ = revertCheckpoint()
					return FaultResult(err, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
				}
				continue
			}

			_ = revertCheckpoint()
			return FaultResult(ErrInvalidOpcode, ctx.GasRemaining-frame.RemainingGas(), frame.RemainingGas(), frame.LastCreatedAddress)
		}
	}
}
