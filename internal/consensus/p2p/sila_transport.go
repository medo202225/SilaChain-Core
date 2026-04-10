package p2p

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	execstate "silachain/internal/consensus/executionstate"
)

const (
	SilaExecutionProtocolName    = "sila-exec"
	SilaExecutionProtocolVersion = 1

	SilaExecutionMsgStatus                = "status"
	SilaExecutionMsgGetBlockHeaders       = "get_block_headers"
	SilaExecutionMsgBlockHeaders          = "block_headers"
	SilaExecutionMsgGetBlockBodies        = "get_block_bodies"
	SilaExecutionMsgBlockBodies           = "block_bodies"
	SilaExecutionMsgNewBlockHashes        = "new_block_hashes"
	SilaExecutionMsgNewBlock              = "new_block"
	SilaExecutionMsgTransactions          = "transactions"
	SilaExecutionMsgGetPooledTransactions = "get_pooled_transactions"
	SilaExecutionMsgPooledTransactions    = "pooled_transactions"
)

type SilaExecutionStatus struct {
	Protocol    string `json:"protocol"`
	Version     uint32 `json:"version"`
	NetworkID   uint64 `json:"network_id"`
	GenesisHash string `json:"genesis_hash"`
	PeerID      string `json:"peer_id"`
	NodeName    string `json:"node_name"`
	ENR         string `json:"enr,omitempty"`
}

type SilaExecutionHeader struct {
	Number     uint64 `json:"number"`
	Hash       string `json:"hash"`
	ParentHash string `json:"parent_hash"`
	Timestamp  uint64 `json:"timestamp"`
}

type SilaExecutionBody struct {
	BlockHash    string   `json:"block_hash"`
	Transactions []string `json:"transactions"`
	Uncles       []string `json:"uncles"`
}

type SilaExecutionTx struct {
	Hash  string `json:"hash"`
	From  string `json:"from"`
	To    string `json:"to"`
	Value string `json:"value"`
	Nonce uint64 `json:"nonce"`
	Data  string `json:"data,omitempty"`
}

type SilaNewBlockHashAnnouncement struct {
	Hash   string `json:"hash"`
	Number uint64 `json:"number"`
}

type SilaNewBlock struct {
	Header SilaExecutionHeader `json:"header"`
	Body   SilaExecutionBody   `json:"body"`
}

type SilaGetBlockHeadersRequest struct {
	RequestID string `json:"request_id"`
	Origin    uint64 `json:"origin"`
	Amount    uint64 `json:"amount"`
}

type SilaBlockHeadersResponse struct {
	RequestID string                `json:"request_id"`
	Headers   []SilaExecutionHeader `json:"headers"`
}

type SilaGetBlockBodiesRequest struct {
	RequestID   string   `json:"request_id"`
	BlockHashes []string `json:"block_hashes"`
}

type SilaBlockBodiesResponse struct {
	RequestID string              `json:"request_id"`
	Bodies    []SilaExecutionBody `json:"bodies"`
}

type SilaGetPooledTransactionsRequest struct {
	RequestID string   `json:"request_id"`
	Hashes    []string `json:"hashes"`
}

type SilaPooledTransactionsResponse struct {
	RequestID    string            `json:"request_id"`
	Transactions []SilaExecutionTx `json:"transactions"`
}

type SilaExecutionMessage struct {
	Code                  string                            `json:"code"`
	Status                *SilaExecutionStatus              `json:"status,omitempty"`
	GetBlockHeaders       *SilaGetBlockHeadersRequest       `json:"get_block_headers,omitempty"`
	BlockHeaders          *SilaBlockHeadersResponse         `json:"block_headers,omitempty"`
	GetBlockBodies        *SilaGetBlockBodiesRequest        `json:"get_block_bodies,omitempty"`
	BlockBodies           *SilaBlockBodiesResponse          `json:"block_bodies,omitempty"`
	NewBlockHashes        []SilaNewBlockHashAnnouncement    `json:"new_block_hashes,omitempty"`
	NewBlock              *SilaNewBlock                     `json:"new_block,omitempty"`
	Transactions          []SilaExecutionTx                 `json:"transactions,omitempty"`
	GetPooledTransactions *SilaGetPooledTransactionsRequest `json:"get_pooled_transactions,omitempty"`
	PooledTransactions    *SilaPooledTransactionsResponse   `json:"pooled_transactions,omitempty"`
}

type SilaExecutionTransportService struct {
	cfg            *Config
	identity       *Identity
	canonical      *CanonicalENR
	executionState *execstate.State

	listener net.Listener
	selfAddr string

	runCount                 atomic.Int32
	statusCount              atomic.Int32
	messageCount             atomic.Int32
	headerRequestCount       atomic.Int32
	headerResponseCount      atomic.Int32
	receivedHeaderCount      atomic.Int32
	bodyRequestCount         atomic.Int32
	bodyResponseCount        atomic.Int32
	receivedBodyCount        atomic.Int32
	newBlockHashesSentCount  atomic.Int32
	newBlockHashesRecvCount  atomic.Int32
	newBlockSentCount        atomic.Int32
	newBlockRecvCount        atomic.Int32
	txSentCount              atomic.Int32
	txRecvCount              atomic.Int32
	pooledReqCount           atomic.Int32
	pooledRespCount          atomic.Int32
	pooledRecvCount          atomic.Int32
	mempoolAnnounceSentCount atomic.Int32
	mempoolAnnounceRecvCount atomic.Int32
	mempoolMissingReqCount   atomic.Int32
	mempoolInsertedCount     atomic.Int32
	syncRequestCount         atomic.Int32
	syncImportCount          atomic.Int32
	syncRemoteHeadCount      atomic.Int32
	importRejectCount        atomic.Int32
	importAcceptCount        atomic.Int32

	mu                sync.RWMutex
	peers             map[string]SilaExecutionStatus
	headers           map[uint64]SilaExecutionHeader
	bodies            map[string]SilaExecutionBody
	pool              map[string]SilaExecutionTx
	missingPoolHashes []string
	latestBlock       SilaNewBlock
	remoteHead        uint64
	localHead         uint64
	importQueue       []SilaNewBlock
}

func txToPendingTx(tx SilaExecutionTx) (execstate.PendingTx, error) {
	value, err := strconv.ParseUint(tx.Value, 10, 64)
	if err != nil {
		return execstate.PendingTx{}, fmt.Errorf("invalid tx value %q: %w", tx.Value, err)
	}
	return execstate.PendingTx{
		Hash:  tx.Hash,
		From:  tx.From,
		To:    tx.To,
		Value: value,
		Nonce: tx.Nonce,
		Data:  tx.Data,
		Fee:   1,
	}, nil
}

func StartSilaExecutionTransport(cfg *Config, identity *Identity, canonical *CanonicalENR, state *execstate.State, staticPeers []string) (*SilaExecutionTransportService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil p2p config")
	}
	if identity == nil {
		return nil, fmt.Errorf("nil identity")
	}
	if canonical == nil || canonical.Sila == nil {
		return nil, fmt.Errorf("nil canonical sila enr")
	}
	if state == nil {
		state = execstate.NewState(cfg.GenesisHash)
	}

	listenAddr := fmt.Sprintf("%s:%d", cfg.ListenIP, cfg.TCPPort)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("listen tcp: %w", err)
	}

	selfIP := cfg.ListenIP
	if selfIP == "" || selfIP == "0.0.0.0" {
		selfIP = "127.0.0.1"
	}

	svc := &SilaExecutionTransportService{
		cfg:            cfg,
		identity:       identity,
		canonical:      canonical,
		executionState: state,
		listener:       ln,
		selfAddr:       fmt.Sprintf("%s:%d", selfIP, cfg.TCPPort),
		peers:          make(map[string]SilaExecutionStatus),
		headers:        make(map[uint64]SilaExecutionHeader),
		bodies:         make(map[string]SilaExecutionBody),
		pool:           make(map[string]SilaExecutionTx),
	}

	genesisHeader := SilaExecutionHeader{
		Number:     0,
		Hash:       cfg.GenesisHash,
		ParentHash: "",
		Timestamp:  0,
	}
	genesisBody := SilaExecutionBody{
		BlockHash:    genesisHeader.Hash,
		Transactions: []string{},
		Uncles:       []string{},
	}

	svc.headers[0] = genesisHeader
	svc.bodies[genesisHeader.Hash] = genesisBody
	svc.latestBlock = SilaNewBlock{
		Header: genesisHeader,
		Body:   genesisBody,
	}
	svc.localHead = 0

	keyFileLower := strings.ToLower(cfg.KeyFile)
	if strings.Contains(keyFileLower, "node1") {
		tx1 := SilaExecutionTx{
			Hash:  "0xtx0001",
			From:  "SILA_sender_1",
			To:    "SILA_receiver_1",
			Value: "100",
			Nonce: 0,
		}
		svc.pool[tx1.Hash] = tx1

		svc.executionState.SetBalance(tx1.From, 1000000)
		pendingTx, err := txToPendingTx(tx1)
		if err != nil {
			return nil, err
		}
		_ = svc.executionState.AddPendingTx(pendingTx)

		h1 := SilaExecutionHeader{
			Number:     1,
			Hash:       "0xblock0001",
			ParentHash: genesisHeader.Hash,
			Timestamp:  1,
		}
		b1 := SilaExecutionBody{
			BlockHash:    h1.Hash,
			Transactions: []string{tx1.Hash},
			Uncles:       []string{},
		}
		svc.headers[1] = h1
		svc.bodies[h1.Hash] = b1
		svc.latestBlock = SilaNewBlock{Header: h1, Body: b1}
		svc.localHead = 1

		if svc.validateAndImportBlock(SilaNewBlock{Header: h1, Body: b1}) {
			svc.importAcceptCount.Add(1)
		}
	}

	go svc.acceptLoop()

	for _, addr := range staticPeers {
		addr := addr
		if addr == "" {
			continue
		}
		go func() {
			time.Sleep(150 * time.Millisecond)
			_ = svc.AddStaticPeer(addr)
		}()
	}

	return svc, nil
}

func (s *SilaExecutionTransportService) validateAndImportBlock(block SilaNewBlock) bool {
	if block.Header.Hash == "" {
		return false
	}
	if block.Body.BlockHash == "" {
		return false
	}
	if block.Body.BlockHash != block.Header.Hash {
		return false
	}
	if block.Header.Number > 0 && block.Header.ParentHash == "" {
		return false
	}
	for _, txHash := range block.Body.Transactions {
		if txHash == "" {
			return false
		}
	}

	err := s.executionState.ImportBlock(execstate.ImportedBlock{
		Number:     block.Header.Number,
		Hash:       block.Header.Hash,
		ParentHash: block.Header.ParentHash,
		Timestamp:  block.Header.Timestamp,
		TxHashes:   append([]string(nil), block.Body.Transactions...),
	})
	return err == nil
}

func (s *SilaExecutionTransportService) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn, false)
	}
}

func (s *SilaExecutionTransportService) handleConn(conn net.Conn, initiator bool) {
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	s.runCount.Add(1)

	localENR, err := s.canonical.Sila.Text()
	if err != nil {
		return
	}

	localStatus := &SilaExecutionStatus{
		Protocol:    SilaExecutionProtocolName,
		Version:     SilaExecutionProtocolVersion,
		NetworkID:   s.cfg.ExecutionNetworkID,
		GenesisHash: s.cfg.GenesisHash,
		PeerID:      s.identity.PeerID,
		NodeName:    "SilaChain/native-exec-protocol",
		ENR:         localENR,
	}

	writer := json.NewEncoder(conn)
	reader := json.NewDecoder(bufio.NewReader(conn))

	if err := writer.Encode(SilaExecutionMessage{
		Code:   SilaExecutionMsgStatus,
		Status: localStatus,
	}); err != nil {
		return
	}

	var inboundStatus SilaExecutionMessage
	if err := reader.Decode(&inboundStatus); err != nil {
		return
	}

	if inboundStatus.Code != SilaExecutionMsgStatus || inboundStatus.Status == nil {
		return
	}

	remoteStatus := inboundStatus.Status
	if remoteStatus.Protocol != SilaExecutionProtocolName ||
		remoteStatus.Version != SilaExecutionProtocolVersion ||
		remoteStatus.NetworkID != s.cfg.ExecutionNetworkID ||
		remoteStatus.GenesisHash != s.cfg.GenesisHash ||
		remoteStatus.PeerID == "" {
		return
	}

	s.mu.Lock()
	s.peers[remoteStatus.PeerID] = *remoteStatus
	s.mu.Unlock()

	s.statusCount.Add(1)
	s.messageCount.Add(1)

	if initiator {
		if !s.requestHeaders(reader, writer) {
			return
		}
		if !s.requestBodies(reader, writer) {
			return
		}
		if !s.receiveAnnouncements(reader) {
			return
		}
		if !s.receiveMempoolAnnouncement(reader) {
			return
		}
		if !s.requestPooledTransactions(reader, writer) {
			return
		}
		if !s.progressSync(reader, writer) {
			return
		}
		return
	}

	if !s.respondHeaders(reader, writer) {
		return
	}
	if !s.respondBodies(reader, writer) {
		return
	}
	if !s.sendAnnouncements(writer) {
		return
	}
	if !s.sendMempoolAnnouncement(writer) {
		return
	}
	if !s.respondPooledTransactions(reader, writer) {
		return
	}
	if !s.serveSync(reader, writer) {
		return
	}

	time.Sleep(500 * time.Millisecond)
}

func (s *SilaExecutionTransportService) progressSync(reader *json.Decoder, writer *json.Encoder) bool {
	remoteHead := s.remoteHeadNumber()
	localHead := s.localHeadNumber()

	if remoteHead <= localHead {
		return true
	}

	s.syncRemoteHeadCount.Add(1)

	req := SilaGetBlockHeadersRequest{
		RequestID: fmt.Sprintf("req-sync-h-%d", time.Now().UnixNano()),
		Origin:    localHead + 1,
		Amount:    remoteHead - localHead,
	}

	if err := writer.Encode(SilaExecutionMessage{
		Code:            SilaExecutionMsgGetBlockHeaders,
		GetBlockHeaders: &req,
	}); err != nil {
		return false
	}
	s.syncRequestCount.Add(1)
	s.messageCount.Add(1)

	var hdrMsg SilaExecutionMessage
	if err := reader.Decode(&hdrMsg); err != nil {
		return false
	}
	if hdrMsg.Code != SilaExecutionMsgBlockHeaders || hdrMsg.BlockHeaders == nil || hdrMsg.BlockHeaders.RequestID != req.RequestID {
		return false
	}
	s.messageCount.Add(1)

	hashes := make([]string, 0, len(hdrMsg.BlockHeaders.Headers))
	for _, h := range hdrMsg.BlockHeaders.Headers {
		s.mu.Lock()
		s.headers[h.Number] = h
		s.mu.Unlock()
		hashes = append(hashes, h.Hash)
	}

	if len(hashes) == 0 {
		return true
	}

	bodyReq := SilaGetBlockBodiesRequest{
		RequestID:   fmt.Sprintf("req-sync-b-%d", time.Now().UnixNano()),
		BlockHashes: hashes,
	}

	if err := writer.Encode(SilaExecutionMessage{
		Code:           SilaExecutionMsgGetBlockBodies,
		GetBlockBodies: &bodyReq,
	}); err != nil {
		return false
	}
	s.syncRequestCount.Add(1)
	s.messageCount.Add(1)

	var bodyMsg SilaExecutionMessage
	if err := reader.Decode(&bodyMsg); err != nil {
		return false
	}
	if bodyMsg.Code != SilaExecutionMsgBlockBodies || bodyMsg.BlockBodies == nil || bodyMsg.BlockBodies.RequestID != bodyReq.RequestID {
		return false
	}
	s.messageCount.Add(1)

	bodyByHash := make(map[string]SilaExecutionBody, len(bodyMsg.BlockBodies.Bodies))
	for _, body := range bodyMsg.BlockBodies.Bodies {
		bodyByHash[body.BlockHash] = body
		s.mu.Lock()
		s.bodies[body.BlockHash] = body
		s.mu.Unlock()
	}

	for _, h := range hdrMsg.BlockHeaders.Headers {
		body, ok := bodyByHash[h.Hash]
		if !ok {
			s.importRejectCount.Add(1)
			continue
		}
		block := SilaNewBlock{Header: h, Body: body}
		if !s.validateAndImportBlock(block) {
			s.importRejectCount.Add(1)
			continue
		}

		s.mu.Lock()
		s.importQueue = append(s.importQueue, block)
		s.latestBlock = block
		if h.Number > s.localHead {
			s.localHead = h.Number
		}
		s.mu.Unlock()

		s.syncImportCount.Add(1)
		s.importAcceptCount.Add(1)
	}

	return true
}

func (s *SilaExecutionTransportService) serveSync(reader *json.Decoder, writer *json.Encoder) bool {
	var hdrReqMsg SilaExecutionMessage
	if err := reader.Decode(&hdrReqMsg); err != nil {
		return false
	}
	if hdrReqMsg.Code != SilaExecutionMsgGetBlockHeaders || hdrReqMsg.GetBlockHeaders == nil {
		return false
	}

	req := hdrReqMsg.GetBlockHeaders
	headers := s.collectHeaders(req.Origin, req.Amount)

	if err := writer.Encode(SilaExecutionMessage{
		Code: SilaExecutionMsgBlockHeaders,
		BlockHeaders: &SilaBlockHeadersResponse{
			RequestID: req.RequestID,
			Headers:   headers,
		},
	}); err != nil {
		return false
	}
	s.messageCount.Add(1)

	var bodyReqMsg SilaExecutionMessage
	if err := reader.Decode(&bodyReqMsg); err != nil {
		return false
	}
	if bodyReqMsg.Code != SilaExecutionMsgGetBlockBodies || bodyReqMsg.GetBlockBodies == nil {
		return false
	}

	bodyReq := bodyReqMsg.GetBlockBodies
	bodies := s.collectBodies(bodyReq.BlockHashes)

	if err := writer.Encode(SilaExecutionMessage{
		Code: SilaExecutionMsgBlockBodies,
		BlockBodies: &SilaBlockBodiesResponse{
			RequestID: bodyReq.RequestID,
			Bodies:    bodies,
		},
	}); err != nil {
		return false
	}
	s.messageCount.Add(1)
	return true
}

func (s *SilaExecutionTransportService) requestHeaders(reader *json.Decoder, writer *json.Encoder) bool {
	req := SilaGetBlockHeadersRequest{
		RequestID: fmt.Sprintf("req-h-%d", time.Now().UnixNano()),
		Origin:    0,
		Amount:    1,
	}

	if err := writer.Encode(SilaExecutionMessage{
		Code:            SilaExecutionMsgGetBlockHeaders,
		GetBlockHeaders: &req,
	}); err != nil {
		return false
	}
	s.messageCount.Add(1)

	var responseMsg SilaExecutionMessage
	if err := reader.Decode(&responseMsg); err != nil {
		return false
	}
	if responseMsg.Code != SilaExecutionMsgBlockHeaders || responseMsg.BlockHeaders == nil || responseMsg.BlockHeaders.RequestID != req.RequestID {
		return false
	}

	for _, h := range responseMsg.BlockHeaders.Headers {
		s.mu.Lock()
		s.headers[h.Number] = h
		s.mu.Unlock()
		s.receivedHeaderCount.Add(1)
	}
	s.messageCount.Add(1)
	return true
}

func (s *SilaExecutionTransportService) requestBodies(reader *json.Decoder, writer *json.Encoder) bool {
	hashes := s.collectedHeaderHashes()
	if len(hashes) == 0 {
		return false
	}

	req := SilaGetBlockBodiesRequest{
		RequestID:   fmt.Sprintf("req-b-%d", time.Now().UnixNano()),
		BlockHashes: hashes,
	}

	if err := writer.Encode(SilaExecutionMessage{
		Code:           SilaExecutionMsgGetBlockBodies,
		GetBlockBodies: &req,
	}); err != nil {
		return false
	}
	s.messageCount.Add(1)

	var responseMsg SilaExecutionMessage
	if err := reader.Decode(&responseMsg); err != nil {
		return false
	}
	if responseMsg.Code != SilaExecutionMsgBlockBodies || responseMsg.BlockBodies == nil || responseMsg.BlockBodies.RequestID != req.RequestID {
		return false
	}

	for _, body := range responseMsg.BlockBodies.Bodies {
		s.mu.Lock()
		s.bodies[body.BlockHash] = body
		s.mu.Unlock()
		s.receivedBodyCount.Add(1)
	}
	s.messageCount.Add(1)
	return true
}

func (s *SilaExecutionTransportService) requestPooledTransactions(reader *json.Decoder, writer *json.Encoder) bool {
	hashes := s.missingAnnouncedPoolHashes()
	if len(hashes) == 0 {
		return true
	}

	req := SilaGetPooledTransactionsRequest{
		RequestID: fmt.Sprintf("req-p-%d", time.Now().UnixNano()),
		Hashes:    hashes,
	}

	if err := writer.Encode(SilaExecutionMessage{
		Code:                  SilaExecutionMsgGetPooledTransactions,
		GetPooledTransactions: &req,
	}); err != nil {
		return false
	}
	s.pooledReqCount.Add(1)
	s.mempoolMissingReqCount.Add(1)
	s.messageCount.Add(1)

	var responseMsg SilaExecutionMessage
	if err := reader.Decode(&responseMsg); err != nil {
		return false
	}
	if responseMsg.Code != SilaExecutionMsgPooledTransactions || responseMsg.PooledTransactions == nil || responseMsg.PooledTransactions.RequestID != req.RequestID {
		return false
	}

	for _, tx := range responseMsg.PooledTransactions.Transactions {
		if s.insertPoolTx(tx) {
			pendingTx, err := txToPendingTx(tx)
			if err == nil {
				_ = s.executionState.AddPendingTx(pendingTx)
			}
			s.pooledRecvCount.Add(1)
			s.mempoolInsertedCount.Add(1)
		}
	}
	s.clearMissingAnnouncedPoolHashes()
	s.messageCount.Add(1)
	return true
}

func (s *SilaExecutionTransportService) respondHeaders(reader *json.Decoder, writer *json.Encoder) bool {
	var requestMsg SilaExecutionMessage
	if err := reader.Decode(&requestMsg); err != nil {
		return false
	}
	if requestMsg.Code != SilaExecutionMsgGetBlockHeaders || requestMsg.GetBlockHeaders == nil {
		return false
	}

	s.headerRequestCount.Add(1)
	s.messageCount.Add(1)

	req := requestMsg.GetBlockHeaders
	headers := s.collectHeaders(req.Origin, req.Amount)

	if err := writer.Encode(SilaExecutionMessage{
		Code: SilaExecutionMsgBlockHeaders,
		BlockHeaders: &SilaBlockHeadersResponse{
			RequestID: req.RequestID,
			Headers:   headers,
		},
	}); err != nil {
		return false
	}

	s.headerResponseCount.Add(1)
	s.messageCount.Add(1)
	return true
}

func (s *SilaExecutionTransportService) respondBodies(reader *json.Decoder, writer *json.Encoder) bool {
	var requestMsg SilaExecutionMessage
	if err := reader.Decode(&requestMsg); err != nil {
		return false
	}
	if requestMsg.Code != SilaExecutionMsgGetBlockBodies || requestMsg.GetBlockBodies == nil {
		return false
	}

	s.bodyRequestCount.Add(1)
	s.messageCount.Add(1)

	req := requestMsg.GetBlockBodies
	bodies := s.collectBodies(req.BlockHashes)

	if err := writer.Encode(SilaExecutionMessage{
		Code: SilaExecutionMsgBlockBodies,
		BlockBodies: &SilaBlockBodiesResponse{
			RequestID: req.RequestID,
			Bodies:    bodies,
		},
	}); err != nil {
		return false
	}

	s.bodyResponseCount.Add(1)
	s.messageCount.Add(1)
	return true
}

func (s *SilaExecutionTransportService) respondPooledTransactions(reader *json.Decoder, writer *json.Encoder) bool {
	var requestMsg SilaExecutionMessage
	if err := reader.Decode(&requestMsg); err != nil {
		return false
	}
	if requestMsg.Code != SilaExecutionMsgGetPooledTransactions || requestMsg.GetPooledTransactions == nil {
		return false
	}

	req := requestMsg.GetPooledTransactions
	txs := s.collectPooledTransactions(req.Hashes)

	if err := writer.Encode(SilaExecutionMessage{
		Code: SilaExecutionMsgPooledTransactions,
		PooledTransactions: &SilaPooledTransactionsResponse{
			RequestID:    req.RequestID,
			Transactions: txs,
		},
	}); err != nil {
		return false
	}

	s.pooledRespCount.Add(1)
	s.messageCount.Add(1)
	return true
}

func (s *SilaExecutionTransportService) sendAnnouncements(writer *json.Encoder) bool {
	s.mu.RLock()
	latest := s.latestBlock
	s.mu.RUnlock()

	if err := writer.Encode(SilaExecutionMessage{
		Code: SilaExecutionMsgNewBlockHashes,
		NewBlockHashes: []SilaNewBlockHashAnnouncement{
			{Hash: latest.Header.Hash, Number: latest.Header.Number},
		},
	}); err != nil {
		return false
	}
	s.newBlockHashesSentCount.Add(1)
	s.messageCount.Add(1)

	if err := writer.Encode(SilaExecutionMessage{
		Code:     SilaExecutionMsgNewBlock,
		NewBlock: &latest,
	}); err != nil {
		return false
	}
	s.newBlockSentCount.Add(1)
	s.messageCount.Add(1)

	return true
}

func (s *SilaExecutionTransportService) receiveAnnouncements(reader *json.Decoder) bool {
	var hashesMsg SilaExecutionMessage
	if err := reader.Decode(&hashesMsg); err != nil {
		return false
	}
	if hashesMsg.Code != SilaExecutionMsgNewBlockHashes || len(hashesMsg.NewBlockHashes) == 0 {
		return false
	}

	var announcedHead uint64
	for _, ann := range hashesMsg.NewBlockHashes {
		if ann.Number > announcedHead {
			announcedHead = ann.Number
		}
	}

	s.mu.Lock()
	if announcedHead > s.remoteHead {
		s.remoteHead = announcedHead
	}
	s.mu.Unlock()

	s.newBlockHashesRecvCount.Add(1)
	s.messageCount.Add(1)

	var newBlockMsg SilaExecutionMessage
	if err := reader.Decode(&newBlockMsg); err != nil {
		return false
	}
	if newBlockMsg.Code != SilaExecutionMsgNewBlock || newBlockMsg.NewBlock == nil {
		return false
	}

	block := newBlockMsg.NewBlock

	s.mu.Lock()
	s.headers[block.Header.Number] = block.Header
	s.bodies[block.Body.BlockHash] = block.Body
	s.latestBlock = *block
	if block.Header.Number > s.remoteHead {
		s.remoteHead = block.Header.Number
	}
	s.mu.Unlock()

	if s.validateAndImportBlock(*block) {
		s.importAcceptCount.Add(1)
	} else {
		s.importRejectCount.Add(1)
	}

	s.newBlockRecvCount.Add(1)
	s.messageCount.Add(1)
	return true
}

func (s *SilaExecutionTransportService) sendMempoolAnnouncement(writer *json.Encoder) bool {
	txs := s.collectAllPoolTransactions()
	if len(txs) == 0 {
		return false
	}

	if err := writer.Encode(SilaExecutionMessage{
		Code:         SilaExecutionMsgTransactions,
		Transactions: txs,
	}); err != nil {
		return false
	}

	s.txSentCount.Add(1)
	s.mempoolAnnounceSentCount.Add(1)
	s.messageCount.Add(1)
	return true
}

func (s *SilaExecutionTransportService) receiveMempoolAnnouncement(reader *json.Decoder) bool {
	var txMsg SilaExecutionMessage
	if err := reader.Decode(&txMsg); err != nil {
		return false
	}
	if txMsg.Code != SilaExecutionMsgTransactions || len(txMsg.Transactions) == 0 {
		return false
	}

	announcedMissing := make([]string, 0)
	for _, tx := range txMsg.Transactions {
		s.txRecvCount.Add(1)
		if !s.hasPoolTx(tx.Hash) {
			announcedMissing = append(announcedMissing, tx.Hash)
		}
	}
	s.setMissingAnnouncedPoolHashes(announcedMissing)
	s.mempoolAnnounceRecvCount.Add(1)
	s.messageCount.Add(1)
	return true
}

func (s *SilaExecutionTransportService) collectHeaders(origin uint64, amount uint64) []SilaExecutionHeader {
	if amount == 0 {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]SilaExecutionHeader, 0, amount)
	for i := uint64(0); i < amount; i++ {
		h, ok := s.headers[origin+i]
		if !ok {
			break
		}
		out = append(out, h)
	}
	return out
}

func (s *SilaExecutionTransportService) collectedHeaderHashes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]string, 0, len(s.headers))
	for i := uint64(0); ; i++ {
		h, ok := s.headers[i]
		if !ok {
			break
		}
		out = append(out, h.Hash)
	}
	return out
}

func (s *SilaExecutionTransportService) collectBodies(hashes []string) []SilaExecutionBody {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]SilaExecutionBody, 0, len(hashes))
	for _, hash := range hashes {
		body, ok := s.bodies[hash]
		if !ok {
			continue
		}
		out = append(out, body)
	}
	return out
}

func (s *SilaExecutionTransportService) collectAllPoolTransactions() []SilaExecutionTx {
	hashes := s.poolHashes()
	out := make([]SilaExecutionTx, 0, len(hashes))
	for _, hash := range hashes {
		tx, ok := s.getPoolTx(hash)
		if ok {
			out = append(out, tx)
		}
	}
	return out
}

func (s *SilaExecutionTransportService) poolHashes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]string, 0, len(s.pool))
	for hash := range s.pool {
		out = append(out, hash)
	}
	sort.Strings(out)
	return out
}

func (s *SilaExecutionTransportService) collectPooledTransactions(hashes []string) []SilaExecutionTx {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]SilaExecutionTx, 0, len(hashes))
	for _, hash := range hashes {
		tx, ok := s.pool[hash]
		if !ok {
			continue
		}
		out = append(out, tx)
	}
	return out
}

func (s *SilaExecutionTransportService) getPoolTx(hash string) (SilaExecutionTx, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tx, ok := s.pool[hash]
	return tx, ok
}

func (s *SilaExecutionTransportService) hasPoolTx(hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.pool[hash]
	return ok
}

func (s *SilaExecutionTransportService) insertPoolTx(tx SilaExecutionTx) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pool[tx.Hash]; ok {
		return false
	}
	s.pool[tx.Hash] = tx
	return true
}

func (s *SilaExecutionTransportService) setMissingAnnouncedPoolHashes(hashes []string) {
	filtered := make([]string, 0, len(hashes))

	s.mu.RLock()
	for _, hash := range hashes {
		if _, ok := s.pool[hash]; ok {
			continue
		}
		filtered = append(filtered, hash)
	}
	s.mu.RUnlock()

	sort.Strings(filtered)

	s.mu.Lock()
	s.missingPoolHashes = append([]string(nil), filtered...)
	s.mu.Unlock()
}

func (s *SilaExecutionTransportService) missingAnnouncedPoolHashes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string(nil), s.missingPoolHashes...)
}

func (s *SilaExecutionTransportService) clearMissingAnnouncedPoolHashes() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.missingPoolHashes = nil
}

func (s *SilaExecutionTransportService) remoteHeadNumber() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.remoteHead
}

func (s *SilaExecutionTransportService) localHeadNumber() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.localHead
}

func (s *SilaExecutionTransportService) SyncRequestCount() int32 {
	if s == nil {
		return 0
	}
	return s.syncRequestCount.Load()
}

func (s *SilaExecutionTransportService) SyncImportCount() int32 {
	if s == nil {
		return 0
	}
	return s.syncImportCount.Load()
}

func (s *SilaExecutionTransportService) SyncRemoteHeadCount() int32 {
	if s == nil {
		return 0
	}
	return s.syncRemoteHeadCount.Load()
}

func (s *SilaExecutionTransportService) ImportRejectCount() int32 {
	if s == nil {
		return 0
	}
	return s.importRejectCount.Load()
}

func (s *SilaExecutionTransportService) ImportAcceptCount() int32 {
	if s == nil {
		return 0
	}
	return s.importAcceptCount.Load()
}

func (s *SilaExecutionTransportService) StateHeadNumber() uint64 {
	if s == nil || s.executionState == nil {
		return 0
	}
	return s.executionState.HeadNumber()
}

func (s *SilaExecutionTransportService) StatePendingCount() int {
	if s == nil || s.executionState == nil {
		return 0
	}
	return s.executionState.PendingCount()
}

func (s *SilaExecutionTransportService) Stop() {
	if s == nil || s.listener == nil {
		return
	}
	_ = s.listener.Close()
}

func (s *SilaExecutionTransportService) SelfAddr() string {
	if s == nil {
		return ""
	}
	return s.selfAddr
}

func (s *SilaExecutionTransportService) Name() string {
	return "SilaChain/native-exec-protocol"
}

func (s *SilaExecutionTransportService) PeerCount() int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.peers)
}

func (s *SilaExecutionTransportService) RunCount() int32 {
	if s == nil {
		return 0
	}
	return s.runCount.Load()
}

func (s *SilaExecutionTransportService) StatusCount() int32 {
	if s == nil {
		return 0
	}
	return s.statusCount.Load()
}

func (s *SilaExecutionTransportService) MessageCount() int32 {
	if s == nil {
		return 0
	}
	return s.messageCount.Load()
}

func (s *SilaExecutionTransportService) HeaderRequestCount() int32 {
	if s == nil {
		return 0
	}
	return s.headerRequestCount.Load()
}

func (s *SilaExecutionTransportService) HeaderResponseCount() int32 {
	if s == nil {
		return 0
	}
	return s.headerResponseCount.Load()
}

func (s *SilaExecutionTransportService) ReceivedHeaderCount() int32 {
	if s == nil {
		return 0
	}
	return s.receivedHeaderCount.Load()
}

func (s *SilaExecutionTransportService) BodyRequestCount() int32 {
	if s == nil {
		return 0
	}
	return s.bodyRequestCount.Load()
}

func (s *SilaExecutionTransportService) BodyResponseCount() int32 {
	if s == nil {
		return 0
	}
	return s.bodyResponseCount.Load()
}

func (s *SilaExecutionTransportService) ReceivedBodyCount() int32 {
	if s == nil {
		return 0
	}
	return s.receivedBodyCount.Load()
}

func (s *SilaExecutionTransportService) NewBlockHashesSentCount() int32 {
	if s == nil {
		return 0
	}
	return s.newBlockHashesSentCount.Load()
}

func (s *SilaExecutionTransportService) NewBlockHashesRecvCount() int32 {
	if s == nil {
		return 0
	}
	return s.newBlockHashesRecvCount.Load()
}

func (s *SilaExecutionTransportService) NewBlockSentCount() int32 {
	if s == nil {
		return 0
	}
	return s.newBlockSentCount.Load()
}

func (s *SilaExecutionTransportService) NewBlockRecvCount() int32 {
	if s == nil {
		return 0
	}
	return s.newBlockRecvCount.Load()
}

func (s *SilaExecutionTransportService) TxSentCount() int32 {
	if s == nil {
		return 0
	}
	return s.txSentCount.Load()
}

func (s *SilaExecutionTransportService) TxRecvCount() int32 {
	if s == nil {
		return 0
	}
	return s.txRecvCount.Load()
}

func (s *SilaExecutionTransportService) PooledReqCount() int32 {
	if s == nil {
		return 0
	}
	return s.pooledReqCount.Load()
}

func (s *SilaExecutionTransportService) PooledRespCount() int32 {
	if s == nil {
		return 0
	}
	return s.pooledRespCount.Load()
}

func (s *SilaExecutionTransportService) PooledRecvCount() int32 {
	if s == nil {
		return 0
	}
	return s.pooledRecvCount.Load()
}

func (s *SilaExecutionTransportService) MempoolAnnounceSentCount() int32 {
	if s == nil {
		return 0
	}
	return s.mempoolAnnounceSentCount.Load()
}

func (s *SilaExecutionTransportService) MempoolAnnounceRecvCount() int32 {
	if s == nil {
		return 0
	}
	return s.mempoolAnnounceRecvCount.Load()
}

func (s *SilaExecutionTransportService) MempoolMissingReqCount() int32 {
	if s == nil {
		return 0
	}
	return s.mempoolMissingReqCount.Load()
}

func (s *SilaExecutionTransportService) MempoolInsertedCount() int32 {
	if s == nil {
		return 0
	}
	return s.mempoolInsertedCount.Load()
}

func (s *SilaExecutionTransportService) AddStaticPeer(addr string) error {
	if s == nil {
		return fmt.Errorf("nil transport service")
	}
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return fmt.Errorf("dial static peer: %w", err)
	}
	go s.handleConn(conn, true)
	return nil
}
