package runtime

import (
	"encoding/json"
	"net/http"
)

type vmExecuteResponse struct {
	Result VMExecuteResult `json:"result"`
	Error  string          `json:"error,omitempty"`
}

func (s *IntrospectionServer) handleVMExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, vmExecuteResponse{Error: "method not allowed"})
		return
	}

	var req VMExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, vmExecuteResponse{Error: err.Error()})
		return
	}

	result, err := s.runtime.VMExecute(req)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, vmExecuteResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, vmExecuteResponse{Result: result})
}
