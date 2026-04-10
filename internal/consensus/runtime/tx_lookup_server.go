package runtime

import "net/http"

type chainTransactionResponse struct {
	Result ChainTransactionResult `json:"result"`
	Error  string                 `json:"error,omitempty"`
}

func (s *IntrospectionServer) handleChainTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainTransactionResponse{Error: "method not allowed"})
		return
	}

	hash := r.URL.Query().Get("hash")
	result, err := s.runtime.ChainTransaction(hash)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainTransactionResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainTransactionResponse{Result: result})
}
