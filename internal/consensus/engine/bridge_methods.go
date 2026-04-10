package engine

import (
	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/forkchoice"
)

func (e *Engine) Assembler() *blockassembly.Assembler {
	if e == nil {
		return nil
	}
	return e.assembler
}

func (e *Engine) ForkchoiceStore() *forkchoice.Store {
	if e == nil {
		return nil
	}
	return e.forkStore
}
