package runtime

import (
	"net/http"
	"strconv"
)

type chainBlockTransactionsResponse struct {
	Result ChainBlockTransactionsResult `json:"result"`
	Error  string                       `json:"error,omitempty"`
}

func (s *IntrospectionServer) handleChainBlockTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainBlockTransactionsResponse{Error: "method not allowed"})
		return
	}

	hash := r.URL.Query().Get("hash")
	result, err := s.runtime.ChainBlockTransactions(hash)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainBlockTransactionsResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainBlockTransactionsResponse{Result: result})
}

func (s *IntrospectionServer) handleChainBlockTransactionsByNumber(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainBlockTransactionsResponse{Error: "method not allowed"})
		return
	}

	number := uint64(0)
	if raw := r.URL.Query().Get("number"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err == nil {
			number = parsed
		}
	}

	result, err := s.runtime.ChainBlockTransactionsByNumber(number)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainBlockTransactionsResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainBlockTransactionsResponse{Result: result})
}
