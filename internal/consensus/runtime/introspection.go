package runtime

import (
	"errors"

	"silachain/internal/consensus/forkchoice"
)

var (
	ErrNilRuntimeIntrospection = errors.New("runtime: nil runtime introspection")
	ErrNilForkchoiceStore      = errors.New("runtime: nil forkchoice store")
)

type ChainHeadResult struct {
	Head forkchoice.BlockRef `json:"head"`
}

type ChainForkchoiceResult struct {
	CanonicalHead forkchoice.BlockRef `json:"canonicalHead"`
	SafeHead      forkchoice.BlockRef `json:"safeHead"`
	FinalizedHead forkchoice.BlockRef `json:"finalizedHead"`
}

type ChainBlockResult struct {
	Block forkchoice.BlockRef `json:"block"`
	Found bool                `json:"found"`
}

type ChainBlocksResult struct {
	Blocks []forkchoice.BlockRef `json:"blocks"`
}

type ChainBlockByNumberResult struct {
	Block forkchoice.BlockRef `json:"block"`
	Found bool                `json:"found"`
}

func (r *Runtime) ChainHead() (ChainHeadResult, error) {
	if r == nil {
		return ChainHeadResult{}, ErrNilRuntimeIntrospection
	}
	if r.engine == nil || r.engine.ForkchoiceStore() == nil {
		return ChainHeadResult{}, ErrNilForkchoiceStore
	}

	head, err := r.engine.ForkchoiceStore().CanonicalHead()
	if err != nil {
		return ChainHeadResult{}, err
	}

	return ChainHeadResult{Head: head}, nil
}

func (r *Runtime) ChainForkchoice() (ChainForkchoiceResult, error) {
	if r == nil {
		return ChainForkchoiceResult{}, ErrNilRuntimeIntrospection
	}
	if r.engine == nil || r.engine.ForkchoiceStore() == nil {
		return ChainForkchoiceResult{}, ErrNilForkchoiceStore
	}

	store := r.engine.ForkchoiceStore()

	canonicalHead, err := store.CanonicalHead()
	if err != nil {
		return ChainForkchoiceResult{}, err
	}
	safeHead, err := store.SafeHead()
	if err != nil {
		return ChainForkchoiceResult{}, err
	}
	finalizedHead, err := store.FinalizedHead()
	if err != nil {
		return ChainForkchoiceResult{}, err
	}

	return ChainForkchoiceResult{
		CanonicalHead: canonicalHead,
		SafeHead:      safeHead,
		FinalizedHead: finalizedHead,
	}, nil
}

func (r *Runtime) ChainBlock(hash string) (ChainBlockResult, error) {
	if r == nil {
		return ChainBlockResult{}, ErrNilRuntimeIntrospection
	}
	if r.engine == nil || r.engine.ForkchoiceStore() == nil {
		return ChainBlockResult{}, ErrNilForkchoiceStore
	}

	block, ok := r.engine.ForkchoiceStore().GetBlock(hash)
	return ChainBlockResult{
		Block: block,
		Found: ok,
	}, nil
}

func (r *Runtime) ChainBlocks(limit int) (ChainBlocksResult, error) {
	if r == nil {
		return ChainBlocksResult{}, ErrNilRuntimeIntrospection
	}
	if r.engine == nil || r.engine.ForkchoiceStore() == nil {
		return ChainBlocksResult{}, ErrNilForkchoiceStore
	}

	blocks, err := r.engine.ForkchoiceStore().CanonicalBlocks(limit)
	if err != nil {
		return ChainBlocksResult{}, err
	}

	return ChainBlocksResult{
		Blocks: blocks,
	}, nil
}

func (r *Runtime) ChainBlockByNumber(number uint64) (ChainBlockByNumberResult, error) {
	if r == nil {
		return ChainBlockByNumberResult{}, ErrNilRuntimeIntrospection
	}
	if r.engine == nil || r.engine.ForkchoiceStore() == nil {
		return ChainBlockByNumberResult{}, ErrNilForkchoiceStore
	}

	block, ok := r.engine.ForkchoiceStore().GetCanonicalBlockByNumber(number)
	return ChainBlockByNumberResult{
		Block: block,
		Found: ok,
	}, nil
}
