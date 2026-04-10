package runtime

import "net/http"

type chainReceiptResponse struct {
	Result ChainReceiptResult `json:"result"`
	Error  string             `json:"error,omitempty"`
}

func (s *IntrospectionServer) handleChainReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, chainReceiptResponse{Error: "method not allowed"})
		return
	}

	txHash := r.URL.Query().Get("txHash")
	result, err := s.runtime.ChainReceipt(txHash)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, chainReceiptResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, chainReceiptResponse{Result: result})
}
