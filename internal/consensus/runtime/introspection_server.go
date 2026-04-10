package runtime

import (
	"errors"
	"net/http"
	"strconv"
)

var (
	ErrNilIntrospectionRuntime = errors.New("runtime: nil introspection runtime")
)

type IntrospectionServer struct {
	runtime *Runtime
	mux     *http.ServeMux
}

type chainHeadResponse struct {
	Result ChainHeadResult `json:"result"`
	Error  string          `json:"error,omitempty"`
}

type chainForkchoiceResponse struct {
	Result ChainForkchoiceResult `json:"result"`
	Error  string                `json:"error,omitempty"`
}

type chainBlockResponse struct {
	Result ChainBlockResult `json:"result"`
	Error  string           `json:"error,omitempty"`
}

type chainBlocksResponse struct {
	Result ChainBlocksResult `json:"result"`
	Error  string            `json:"error,omitempty"`
}

type chainBlockByNumberResponse struct {
	Result ChainBlockByNumberResult `json:"result"`
	Error  string                   `json:"error,omitempty"`
}

func NewIntrospectionServer(rt *Runtime) (*IntrospectionServer, error) {
	if rt == nil {
		return nil, ErrNilIntrospectionRuntime
	}

	s := &IntrospectionServer{
		runtime: rt,
		mux:     http.NewServeMux(),
	}

	s.mux.HandleFunc("/chain/head", s.handleChainHead)
	s.mux.HandleFunc("/chain/forkchoice", s.handleChainForkchoice)
	s.mux.HandleFunc("/chain/block", s.handleChainBlock)
	s.mux.HandleFunc("/chain/blocks", s.handleChainBlocks)
	s.mux.HandleFunc("/chain/blockByNumber", s.handleChainBlockByNumber)
	s.mux.HandleFunc("/chain/tx", s.handleChainTransaction)
	s.mux.HandleFunc("/chain/receipt", s.handleChainReceipt)
	s.mux.HandleFunc("/chain/receiptsByBlock", s.handleChainReceiptsByBlock)
	s.mux.HandleFunc("/chain/receiptsByBlockNumber", s.handleChainReceiptsByBlockNumber)
	s.mux.HandleFunc("/chain/logs", s.handleChainLogs)
	s.mux.HandleFunc("/chain/logsByBlock", s.handleChainLogsByBlock)
	s.mux.HandleFunc("/chain/logsByBlockNumber", s.handleChainLogsByBlockNumber)
	s.mux.HandleFunc("/chain/blockTxs", s.handleChainBlockTransactions)
	s.mux.HandleFunc("/chain/blockTxsByNumber", s.handleChainBlockTransactionsByNumber)
	s.mux.HandleFunc("/state/account", s.handleStateAccount)
	s.mux.HandleFunc("/state/code", s.handleStateCode)
	s.mux.HandleFunc("/state/storage", s.handleStateStorage)
	s.mux.HandleFunc("/vm/execute", s.handleVMExecute)

	return s, nil
}

func (s *IntrospectionServer) Handler() http.Handler {
	if s == nil {
		return http.NewServeMux()
	}
	return s.mux
}

func (s *IntrospectionServer) handleChainHead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainHeadResponse{Error: "method not allowed"})
		return
	}

	result, err := s.runtime.ChainHead()
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainHeadResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainHeadResponse{Result: result})
}

func (s *IntrospectionServer) handleChainForkchoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainForkchoiceResponse{Error: "method not allowed"})
		return
	}

	result, err := s.runtime.ChainForkchoice()
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainForkchoiceResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainForkchoiceResponse{Result: result})
}

func (s *IntrospectionServer) handleChainBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainBlockResponse{Error: "method not allowed"})
		return
	}

	hash := r.URL.Query().Get("hash")
	result, err := s.runtime.ChainBlock(hash)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainBlockResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainBlockResponse{Result: result})
}

func (s *IntrospectionServer) handleChainBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainBlocksResponse{Error: "method not allowed"})
		return
	}

	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err == nil {
			limit = parsed
		}
	}

	result, err := s.runtime.ChainBlocks(limit)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainBlocksResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainBlocksResponse{Result: result})
}

func (s *IntrospectionServer) handleChainBlockByNumber(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainBlockByNumberResponse{Error: "method not allowed"})
		return
	}

	number := uint64(0)
	if raw := r.URL.Query().Get("number"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err == nil {
			number = parsed
		}
	}

	result, err := s.runtime.ChainBlockByNumber(number)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainBlockByNumberResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainBlockByNumberResponse{Result: result})
}
