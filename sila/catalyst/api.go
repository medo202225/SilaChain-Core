package catalyst

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	beaconengine "silachain/beacon/engine"
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

// ForkchoiceUpdatedV1 mirrors the first engine API forkchoice entrypoint.
// V1 allows neither withdrawals nor beacon root.
func (api *ConsensusAPI) ForkchoiceUpdatedV1(ctx context.Context, update beaconengine.ForkchoiceStateV1, payloadAttributes *beaconengine.PayloadAttributes) (beaconengine.ForkChoiceResponse, error) {
	if payloadAttributes != nil {
		switch {
		case payloadAttributes.Withdrawals != nil || payloadAttributes.BeaconRoot != nil:
			return beaconengine.STATUS_INVALID, paramsErr("withdrawals and beacon root not supported in V1")
		case !api.checkFork(payloadAttributes.Timestamp, PayloadForkParis, PayloadForkShanghai):
			return beaconengine.STATUS_INVALID, paramsErr("fcuV1 called post-shanghai")
		}
	}
	return api.forkchoiceUpdated(ctx, update, payloadAttributes, beaconengine.PayloadV1, false)
}

// ForkchoiceUpdatedV2 allows withdrawals on/after shanghai, but not beacon root.
func (api *ConsensusAPI) ForkchoiceUpdatedV2(ctx context.Context, update beaconengine.ForkchoiceStateV1, params *beaconengine.PayloadAttributes) (beaconengine.ForkChoiceResponse, error) {
	if params != nil {
		switch {
		case params.BeaconRoot != nil:
			return beaconengine.STATUS_INVALID, attributesErr("unexpected beacon root")
		case api.checkFork(params.Timestamp, PayloadForkParis) && params.Withdrawals != nil:
			return beaconengine.STATUS_INVALID, attributesErr("withdrawals before shanghai")
		case api.checkFork(params.Timestamp, PayloadForkShanghai) && params.Withdrawals == nil:
			return beaconengine.STATUS_INVALID, attributesErr("missing withdrawals")
		case !api.checkFork(params.Timestamp, PayloadForkParis, PayloadForkShanghai):
			return beaconengine.STATUS_INVALID, unsupportedForkErr("fcuV2 must only be called with paris or shanghai payloads")
		}
	}
	return api.forkchoiceUpdated(ctx, update, params, beaconengine.PayloadV2, false)
}

// ForkchoiceUpdatedV3 requires withdrawals + beacon root.
func (api *ConsensusAPI) ForkchoiceUpdatedV3(ctx context.Context, update beaconengine.ForkchoiceStateV1, params *beaconengine.PayloadAttributes) (beaconengine.ForkChoiceResponse, error) {
	if params != nil {
		switch {
		case params.Withdrawals == nil:
			return beaconengine.STATUS_INVALID, attributesErr("missing withdrawals")
		case params.BeaconRoot == nil:
			return beaconengine.STATUS_INVALID, attributesErr("missing beacon root")
		case !api.checkFork(params.Timestamp, PayloadForkCancun, PayloadForkPrague, PayloadForkOsaka, PayloadForkBPO1, PayloadForkBPO2, PayloadForkBPO3, PayloadForkBPO4, PayloadForkBPO5):
			return beaconengine.STATUS_INVALID, unsupportedForkErr("fcuV3 must only be called for cancun/prague/osaka payloads")
		}
	}
	return api.forkchoiceUpdated(ctx, update, params, beaconengine.PayloadV3, false)
}

// ForkchoiceUpdatedV4 requires withdrawals + beacon root + slot number.
func (api *ConsensusAPI) ForkchoiceUpdatedV4(ctx context.Context, update beaconengine.ForkchoiceStateV1, params *beaconengine.PayloadAttributes) (beaconengine.ForkChoiceResponse, error) {
	if params != nil {
		switch {
		case params.Withdrawals == nil:
			return beaconengine.STATUS_INVALID, attributesErr("missing withdrawals")
		case params.BeaconRoot == nil:
			return beaconengine.STATUS_INVALID, attributesErr("missing beacon root")
		case params.SlotNumber == nil:
			return beaconengine.STATUS_INVALID, attributesErr("missing slot number")
		case !api.checkFork(params.Timestamp, PayloadForkAmsterdam):
			return beaconengine.STATUS_INVALID, unsupportedForkErr("fcuV4 must only be called for amsterdam payloads")
		}
	}
	return api.forkchoiceUpdated(ctx, update, params, beaconengine.PayloadV4, false)
}

func (api *ConsensusAPI) forkchoiceUpdated(ctx context.Context, update beaconengine.ForkchoiceStateV1, payloadAttributes *beaconengine.PayloadAttributes, payloadVersion beaconengine.PayloadVersion, payloadWitness bool) (beaconengine.ForkChoiceResponse, error) {
	_ = ctx
	_ = payloadAttributes
	_ = payloadVersion
	_ = payloadWitness

	api.forkchoiceLock.Lock()
	defer api.forkchoiceLock.Unlock()

	if update.HeadBlockHash == "" {
		return beaconengine.STATUS_INVALID, nil
	}
	api.lastForkchoiceUpdate.Store(time.Now().Unix())

	latestValid := update.HeadBlockHash
	return beaconengine.ForkChoiceResponse{
		PayloadStatus: beaconengine.PayloadStatusV1{
			Status:          beaconengine.VALID,
			LatestValidHash: &latestValid,
		},
		PayloadID: nil,
	}, nil
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

type PayloadFork string

const (
	PayloadForkParis     PayloadFork = "paris"
	PayloadForkShanghai  PayloadFork = "shanghai"
	PayloadForkCancun    PayloadFork = "cancun"
	PayloadForkPrague    PayloadFork = "prague"
	PayloadForkOsaka     PayloadFork = "osaka"
	PayloadForkBPO1      PayloadFork = "bpo1"
	PayloadForkBPO2      PayloadFork = "bpo2"
	PayloadForkBPO3      PayloadFork = "bpo3"
	PayloadForkBPO4      PayloadFork = "bpo4"
	PayloadForkBPO5      PayloadFork = "bpo5"
	PayloadForkAmsterdam PayloadFork = "amsterdam"
)

func (api *ConsensusAPI) checkFork(timestamp uint64, forks ...PayloadFork) bool {
	latest := latestFork(timestamp)
	for _, fork := range forks {
		if latest == fork {
			return true
		}
	}
	return false
}

func latestFork(timestamp uint64) PayloadFork {
	switch {
	case timestamp >= 500:
		return PayloadForkAmsterdam
	case timestamp >= 400:
		return PayloadForkCancun
	case timestamp >= 300:
		return PayloadForkShanghai
	default:
		return PayloadForkParis
	}
}

func paramsErr(msg string) error {
	return beaconengine.InvalidParams.With(errors.New(msg))
}

func attributesErr(msg string) error {
	return beaconengine.InvalidPayloadAttributes.With(errors.New(msg))
}

func unsupportedForkErr(msg string) error {
	return beaconengine.UnsupportedFork.With(errors.New(msg))
}
