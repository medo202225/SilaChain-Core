package vm

func GasCost(op byte) uint64 {
	switch op {
	case OpStop:
		return 0
	case OpAdd, OpSub, OpLT, OpGT, OpEQ, OpIsZero:
		return 3
	case OpAddress, OpCaller, OpCallValue, OpCallDataLoad, OpCallDataSize:
		return 2
	case OpPop:
		return 2
	case OpMLoad, OpMStore:
		return 3
	case OpSLoad:
		return 50
	case OpSStore:
		return 200
	case OpJump, OpJumpI, OpJumpDest:
		return 8
	case OpLog0:
		return 20
	case OpLog1:
		return 30
	case OpLog2:
		return 40
	case OpLog3:
		return 50
	case OpLog4:
		return 60
	case OpCreate:
		return 320
	case OpCall, OpStaticCall:
		return 40
	case OpReturn, OpRevert:
		return 0
	default:
		if IsPush(op) {
			return 3
		}
		return 0
	}
}
