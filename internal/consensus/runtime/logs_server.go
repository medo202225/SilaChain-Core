package runtime

import (
	"net/http"
	"strconv"
)

type chainLogsResponse struct {
	Result ChainLogsResult `json:"result"`
	Error  string          `json:"error,omitempty"`
}

func (s *IntrospectionServer) handleChainLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainLogsResponse{Error: "method not allowed"})
		return
	}

	txHash := r.URL.Query().Get("txHash")
	result, err := s.runtime.ChainLogs(txHash)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainLogsResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainLogsResponse{Result: result})
}

func (s *IntrospectionServer) handleChainLogsByBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainLogsResponse{Error: "method not allowed"})
		return
	}

	hash := r.URL.Query().Get("hash")
	result, err := s.runtime.ChainLogsByBlock(hash)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainLogsResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainLogsResponse{Result: result})
}

func (s *IntrospectionServer) handleChainLogsByBlockNumber(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainLogsResponse{Error: "method not allowed"})
		return
	}

	number := uint64(0)
	if raw := r.URL.Query().Get("number"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err == nil {
			number = parsed
		}
	}

	result, err := s.runtime.ChainLogsByBlockNumber(number)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainLogsResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainLogsResponse{Result: result})
}
