package runtime

import "net/http"

type stateAccountResponse struct {
	Result AccountResult `json:"result"`
	Error  string        `json:"error,omitempty"`
}

type stateCodeResponse struct {
	Result StateCodeResult `json:"result"`
	Error  string          `json:"error,omitempty"`
}

type stateStorageResponse struct {
	Result StateStorageResult `json:"result"`
	Error  string             `json:"error,omitempty"`
}

func (s *IntrospectionServer) handleStateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, stateAccountResponse{Error: "method not allowed"})
		return
	}

	address := r.URL.Query().Get("address")
	result, err := s.runtime.StateAccount(address)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, stateAccountResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, stateAccountResponse{Result: result})
}

func (s *IntrospectionServer) handleStateCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, stateCodeResponse{Error: "method not allowed"})
		return
	}

	address := r.URL.Query().Get("address")
	result, err := s.runtime.StateCode(address)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, stateCodeResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, stateCodeResponse{Result: result})
}

func (s *IntrospectionServer) handleStateStorage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, stateStorageResponse{Error: "method not allowed"})
		return
	}

	address := r.URL.Query().Get("address")
	key := r.URL.Query().Get("key")
	result, err := s.runtime.StateStorage(address, key)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, stateStorageResponse{Result: result, Error: err.Error()})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, stateStorageResponse{Result: result})
}
