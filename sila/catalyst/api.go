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

type storedPayload struct {
	envelope *beaconengine.ExecutionPayloadEnvelope
	full     bool
}

type payloadQueue struct {
	items map[string]storedPayload
}

func newPayloadQueue() *payloadQueue {
	return &payloadQueue{
		items: make(map[string]storedPayload),
	}
}

func (q *payloadQueue) has(id beaconengine.PayloadID) bool {
	if q == nil {
		return false
	}
	_, ok := q.items[id.String()]
	return ok
}

func (q *payloadQueue) put(id beaconengine.PayloadID, env *beaconengine.ExecutionPayloadEnvelope, full bool) {
	if q == nil {
		return
	}
	q.items[id.String()] = storedPayload{
		envelope: env,
		full:     full,
	}
}

func (q *payloadQueue) get(id beaconengine.PayloadID, full bool) *beaconengine.ExecutionPayloadEnvelope {
	if q == nil {
		return nil
	}
	item, ok := q.items[id.String()]
	if !ok {
		return nil
	}
	if full && !item.full {
		return nil
	}
	return item.envelope
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

var invalidStatus = beaconengine.PayloadStatusV1{Status: beaconengine.INVALID}

func (api *ConsensusAPI) NewPayloadV1(ctx context.Context, params beaconengine.ExecutableData) (beaconengine.PayloadStatusV1, error) {
	if params.Withdrawals != nil {
		return invalidStatus, paramsErr("withdrawals not supported in V1")
	}
	return api.newPayload(ctx, params, nil, nil, nil, false)
}

func (api *ConsensusAPI) NewPayloadV2(ctx context.Context, params beaconengine.ExecutableData) (beaconengine.PayloadStatusV1, error) {
	switch {
	case latestFork(params.Timestamp) == PayloadForkCancun || latestFork(params.Timestamp) == PayloadForkAmsterdam:
		return invalidStatus, paramsErr("can't use newPayloadV2 post-cancun")
	case latestFork(params.Timestamp) == PayloadForkShanghai && params.Withdrawals == nil:
		return invalidStatus, paramsErr("nil withdrawals post-shanghai")
	case latestFork(params.Timestamp) == PayloadForkParis && params.Withdrawals != nil:
		return invalidStatus, paramsErr("non-nil withdrawals pre-shanghai")
	case params.ExcessBlobGas != nil:
		return invalidStatus, paramsErr("non-nil excessBlobGas pre-cancun")
	case params.BlobGasUsed != nil:
		return invalidStatus, paramsErr("non-nil blobGasUsed pre-cancun")
	}
	return api.newPayload(ctx, params, nil, nil, nil, false)
}

func (api *ConsensusAPI) NewPayloadV3(ctx context.Context, params beaconengine.ExecutableData, versionedHashes []string, beaconRoot *string) (beaconengine.PayloadStatusV1, error) {
	switch {
	case params.Withdrawals == nil:
		return invalidStatus, paramsErr("nil withdrawals post-shanghai")
	case params.ExcessBlobGas == nil:
		return invalidStatus, paramsErr("nil excessBlobGas post-cancun")
	case params.BlobGasUsed == nil:
		return invalidStatus, paramsErr("nil blobGasUsed post-cancun")
	case versionedHashes == nil:
		return invalidStatus, paramsErr("nil versionedHashes post-cancun")
	case beaconRoot == nil:
		return invalidStatus, paramsErr("nil beaconRoot post-cancun")
	case !api.checkFork(params.Timestamp, PayloadForkCancun):
		return invalidStatus, unsupportedForkErr("newPayloadV3 must only be called for cancun payloads")
	}
	return api.newPayload(ctx, params, versionedHashes, beaconRoot, nil, false)
}

func (api *ConsensusAPI) NewPayloadV4(ctx context.Context, params beaconengine.ExecutableData, versionedHashes []string, beaconRoot *string, executionRequests [][]byte) (beaconengine.PayloadStatusV1, error) {
	switch {
	case params.Withdrawals == nil:
		return invalidStatus, paramsErr("nil withdrawals post-shanghai")
	case params.ExcessBlobGas == nil:
		return invalidStatus, paramsErr("nil excessBlobGas post-cancun")
	case params.BlobGasUsed == nil:
		return invalidStatus, paramsErr("nil blobGasUsed post-cancun")
	case versionedHashes == nil:
		return invalidStatus, paramsErr("nil versionedHashes post-cancun")
	case beaconRoot == nil:
		return invalidStatus, paramsErr("nil beaconRoot post-cancun")
	case executionRequests == nil:
		return invalidStatus, paramsErr("nil executionRequests post-prague")
	case !api.checkFork(params.Timestamp, PayloadForkPrague, PayloadForkOsaka, PayloadForkBPO1, PayloadForkBPO2, PayloadForkBPO3, PayloadForkBPO4, PayloadForkBPO5):
		return invalidStatus, unsupportedForkErr("newPayloadV4 must only be called for prague/osaka payloads")
	}
	if err := validateRequests(executionRequests); err != nil {
		return beaconengine.PayloadStatusV1{Status: beaconengine.INVALID}, beaconengine.InvalidParams.With(err)
	}
	return api.newPayload(ctx, params, versionedHashes, beaconRoot, executionRequests, false)
}

func (api *ConsensusAPI) NewPayloadV5(ctx context.Context, params beaconengine.ExecutableData, versionedHashes []string, beaconRoot *string, executionRequests [][]byte) (beaconengine.PayloadStatusV1, error) {
	switch {
	case params.Withdrawals == nil:
		return invalidStatus, paramsErr("nil withdrawals post-shanghai")
	case params.ExcessBlobGas == nil:
		return invalidStatus, paramsErr("nil excessBlobGas post-cancun")
	case params.BlobGasUsed == nil:
		return invalidStatus, paramsErr("nil blobGasUsed post-cancun")
	case versionedHashes == nil:
		return invalidStatus, paramsErr("nil versionedHashes post-cancun")
	case beaconRoot == nil:
		return invalidStatus, paramsErr("nil beaconRoot post-cancun")
	case executionRequests == nil:
		return invalidStatus, paramsErr("nil executionRequests post-prague")
	case params.SlotNumber == nil:
		return invalidStatus, paramsErr("nil slotnumber post-amsterdam")
	case !api.checkFork(params.Timestamp, PayloadForkAmsterdam):
		return invalidStatus, unsupportedForkErr("newPayloadV5 must only be called for amsterdam payloads")
	}
	if err := validateRequests(executionRequests); err != nil {
		return beaconengine.PayloadStatusV1{Status: beaconengine.INVALID}, beaconengine.InvalidParams.With(err)
	}
	return api.newPayload(ctx, params, versionedHashes, beaconRoot, executionRequests, false)
}

func (api *ConsensusAPI) newPayload(ctx context.Context, params beaconengine.ExecutableData, versionedHashes []string, beaconRoot *string, requests [][]byte, witness bool) (beaconengine.PayloadStatusV1, error) {
	_ = ctx
	_ = versionedHashes
	_ = beaconRoot
	_ = requests
	_ = witness

	api.newPayloadLock.Lock()
	defer api.newPayloadLock.Unlock()

	api.lastNewPayloadUpdate.Store(time.Now().Unix())

	hash := params.BlockHash
	if hash == "" {
		return beaconengine.PayloadStatusV1{
			Status: beaconengine.INVALID,
		}, paramsErr("empty block hash")
	}

	id := beaconengine.PayloadID{byte(beaconengine.PayloadV1), 1, 2, 3, 4, 5, 6, 7}
	if params.SlotNumber != nil {
		id[0] = byte(beaconengine.PayloadV4)
	} else if beaconRoot != nil {
		id[0] = byte(beaconengine.PayloadV3)
	} else if params.Withdrawals != nil {
		id[0] = byte(beaconengine.PayloadV2)
	}

	env := &beaconengine.ExecutionPayloadEnvelope{
		ExecutionPayload: &params,
		BlockValue:       0,
		Override:         false,
		Requests:         requests,
	}
	api.localBlocks.put(id, env, false)

	return beaconengine.PayloadStatusV1{
		Status:          beaconengine.VALID,
		LatestValidHash: &hash,
	}, nil
}

func (api *ConsensusAPI) GetPayloadV1(payloadID beaconengine.PayloadID) (*beaconengine.ExecutableData, error) {
	data, err := api.getPayload(payloadID, false, []beaconengine.PayloadVersion{beaconengine.PayloadV1})
	if err != nil {
		return nil, err
	}
	return data.ExecutionPayload, nil
}

func (api *ConsensusAPI) GetPayloadV2(payloadID beaconengine.PayloadID) (*beaconengine.ExecutionPayloadEnvelope, error) {
	return api.getPayload(payloadID, false, []beaconengine.PayloadVersion{beaconengine.PayloadV1, beaconengine.PayloadV2})
}

func (api *ConsensusAPI) GetPayloadV3(payloadID beaconengine.PayloadID) (*beaconengine.ExecutionPayloadEnvelope, error) {
	return api.getPayload(payloadID, false, []beaconengine.PayloadVersion{beaconengine.PayloadV3})
}

func (api *ConsensusAPI) GetPayloadV4(payloadID beaconengine.PayloadID) (*beaconengine.ExecutionPayloadEnvelope, error) {
	return api.getPayload(payloadID, false, []beaconengine.PayloadVersion{beaconengine.PayloadV3})
}

func (api *ConsensusAPI) GetPayloadV5(payloadID beaconengine.PayloadID) (*beaconengine.ExecutionPayloadEnvelope, error) {
	return api.getPayload(payloadID, false, []beaconengine.PayloadVersion{beaconengine.PayloadV3})
}

func (api *ConsensusAPI) GetPayloadV6(payloadID beaconengine.PayloadID) (*beaconengine.ExecutionPayloadEnvelope, error) {
	return api.getPayload(payloadID, false, []beaconengine.PayloadVersion{beaconengine.PayloadV4})
}

func (api *ConsensusAPI) getPayload(payloadID beaconengine.PayloadID, full bool, versions []beaconengine.PayloadVersion) (*beaconengine.ExecutionPayloadEnvelope, error) {
	if versions != nil && !payloadID.Is(versions...) {
		return nil, beaconengine.UnsupportedFork
	}
	data := api.localBlocks.get(payloadID, full)
	if data == nil {
		return nil, beaconengine.UnknownPayload
	}
	return data, nil
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

func validateRequests(requests [][]byte) error {
	for i, req := range requests {
		if len(req) < 2 {
			return errors.New("empty request")
		}
		if i > 0 && req[0] <= requests[i-1][0] {
			return errors.New("invalid request order")
		}
	}
	return nil
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

func (api *ConsensusAPI) ExchangeCapabilities([]string) []string {
	return []string{
		"engine_forkchoiceUpdatedV1",
		"engine_forkchoiceUpdatedV2",
		"engine_forkchoiceUpdatedV3",
		"engine_forkchoiceUpdatedV4",
		"engine_newPayloadV1",
		"engine_newPayloadV2",
		"engine_newPayloadV3",
		"engine_newPayloadV4",
		"engine_newPayloadV5",
		"engine_getPayloadV1",
		"engine_getPayloadV2",
		"engine_getPayloadV3",
		"engine_getPayloadV4",
		"engine_getPayloadV5",
		"engine_getPayloadV6",
		"engine_getClientVersionV1",
	}
}

func (api *ConsensusAPI) GetClientVersionV1(info beaconengine.ClientVersionV1) []beaconengine.ClientVersionV1 {
	_ = info
	return []beaconengine.ClientVersionV1{
		{
			Code:    beaconengine.ClientCode,
			Name:    beaconengine.ClientName,
			Version: "sila/1.0.0",
			Commit:  "0x00000000",
		},
	}
}
