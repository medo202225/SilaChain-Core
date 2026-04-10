package engineapi

import (
	"errors"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/forkchoice"
)

var (
	ErrNilEngineBridge = errors.New("engineapi: nil engine bridge")
)

type EngineBridge interface {
	Assembler() *blockassembly.Assembler
	ForkchoiceStore() *forkchoice.Store
}

func NewBuilderServiceFromEngine(eng EngineBridge) (*BuilderService, error) {
	if eng == nil {
		return nil, ErrNilEngineBridge
	}

	store := eng.ForkchoiceStore()
	if store == nil {
		return nil, ErrNilStore
	}

	builder := eng.Assembler()
	if builder == nil {
		return nil, ErrNilBuilder
	}

	return NewBuilderService(store, builder)
}
