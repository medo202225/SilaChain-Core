package runtime

import (
	"net/http"
	"strconv"
)

type chainReceiptsByBlockResponse struct {
	Result ChainReceiptsByBlockResult `json:"result"`
	Error  string                     `json:"error,omitempty"`
}

func (s *IntrospectionServer) handleChainReceiptsByBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainReceiptsByBlockResponse{Error: "method not allowed"})
		return
	}

	hash := r.URL.Query().Get("hash")
	result, err := s.runtime.ChainReceiptsByBlock(hash)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainReceiptsByBlockResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainReceiptsByBlockResponse{Result: result})
}

func (s *IntrospectionServer) handleChainReceiptsByBlockNumber(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainReceiptsByBlockResponse{Error: "method not allowed"})
		return
	}

	number := uint64(0)
	if raw := r.URL.Query().Get("number"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err == nil {
			number = parsed
		}
	}

	result, err := s.runtime.ChainReceiptsByBlockNumber(number)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainReceiptsByBlockResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainReceiptsByBlockResponse{Result: result})
}
