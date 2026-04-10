package vm

const (
	OpStop         byte = 0x00
	OpAdd          byte = 0x01
	OpSub          byte = 0x03
	OpLT           byte = 0x10
	OpGT           byte = 0x11
	OpEQ           byte = 0x14
	OpIsZero       byte = 0x15
	OpAddress      byte = 0x30
	OpCaller       byte = 0x33
	OpCallValue    byte = 0x34
	OpCallDataLoad byte = 0x35
	OpCallDataSize byte = 0x36
	OpPop          byte = 0x50
	OpMLoad        byte = 0x51
	OpMStore       byte = 0x52
	OpSLoad        byte = 0x54
	OpSStore       byte = 0x55
	OpJump         byte = 0x56
	OpJumpI        byte = 0x57
	OpJumpDest     byte = 0x5b
	OpPush1        byte = 0x60
	OpLog0         byte = 0xa0
	OpLog1         byte = 0xa1
	OpLog2         byte = 0xa2
	OpLog3         byte = 0xa3
	OpLog4         byte = 0xa4
	OpCreate       byte = 0xf0
	OpCall         byte = 0xf1
	OpReturn       byte = 0xf3
	OpStaticCall   byte = 0xfa
	OpRevert       byte = 0xfd
)

func IsPush(op byte) bool {
	return op >= OpPush1 && op <= 0x7f
}

func PushSize(op byte) uint64 {
	if !IsPush(op) {
		return 0
	}
	return uint64(op - OpPush1 + 1)
}
