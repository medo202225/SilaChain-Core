package catalyst

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	invalidBlockHitEviction      = 128
	invalidTipsetsCap            = 512
	beaconUpdateStartupTimeout   = 30 * time.Second
	beaconUpdateConsensusTimeout = 2 * time.Minute
	beaconUpdateWarnFrequency    = 5 * time.Minute
)

type Backend interface {
	SetBadBlockCallback(func(invalidHash string, originHash string))
}

type Node interface {
	RegisterAPIs([]API)
}

type API struct {
	Namespace     string
	Service       interface{}
	Authenticated bool
}

type headerQueue struct{}

func newHeaderQueue() *headerQueue {
	return &headerQueue{}
}

type payloadQueue struct{}

func newPayloadQueue() *payloadQueue {
	return &payloadQueue{}
}

type ConsensusAPI struct {
	backend Backend

	remoteBlocks *headerQueue
	localBlocks  *payloadQueue

	invalidBlocksHits map[string]int
	invalidTipsets    map[string]string
	invalidLock       sync.Mutex

	lastTransitionUpdate atomic.Int64
	lastForkchoiceUpdate atomic.Int64
	lastNewPayloadUpdate atomic.Int64

	forkchoiceLock sync.Mutex
	newPayloadLock sync.Mutex
}

func Register(stack Node, backend Backend) error {
	stack.RegisterAPIs([]API{
		{
			Namespace:     "engine",
			Service:       NewConsensusAPI(backend),
			Authenticated: true,
		},
	})
	return nil
}

func NewConsensusAPI(backend Backend) *ConsensusAPI {
	api := newConsensusAPIWithoutHeartbeat(backend)
	go api.heartbeat()
	return api
}

func newConsensusAPIWithoutHeartbeat(backend Backend) *ConsensusAPI {
	api := &ConsensusAPI{
		backend:           backend,
		remoteBlocks:      newHeaderQueue(),
		localBlocks:       newPayloadQueue(),
		invalidBlocksHits: make(map[string]int),
		invalidTipsets:    make(map[string]string),
	}
	if backend != nil {
		backend.SetBadBlockCallback(api.setInvalidAncestor)
	}
	return api
}

func (api *ConsensusAPI) setInvalidAncestor(invalidHash string, originHash string) {
	api.invalidLock.Lock()
	defer api.invalidLock.Unlock()

	api.invalidTipsets[originHash] = invalidHash
	api.invalidBlocksHits[invalidHash]++
}

func (api *ConsensusAPI) heartbeat() {
	time.Sleep(beaconUpdateStartupTimeout)

	var offlineLogged time.Time

	for {
		time.Sleep(5 * time.Second)

		lastForkchoiceUpdate := time.Unix(api.lastForkchoiceUpdate.Load(), 0)
		lastNewPayloadUpdate := time.Unix(api.lastNewPayloadUpdate.Load(), 0)

		if time.Since(lastForkchoiceUpdate) <= beaconUpdateConsensusTimeout || time.Since(lastNewPayloadUpdate) <= beaconUpdateConsensusTimeout {
			offlineLogged = time.Time{}
			continue
		}
		if time.Since(offlineLogged) > beaconUpdateWarnFrequency {
			offlineLogged = time.Now()
		}
	}
}
