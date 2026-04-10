package runtime

import (
	"encoding/json"
	"errors"
	"net/http"
)

var (
	ErrNilRuntimeForServer = errors.New("runtime: nil runtime for produce block server")
)

type ProduceBlockServer struct {
	runtime *Runtime
	mux     *http.ServeMux
}

type produceBlockResponse struct {
	Result ProduceBlockResult `json:"result"`
	Error  string             `json:"error,omitempty"`
}

func NewProduceBlockServer(rt *Runtime) (*ProduceBlockServer, error) {
	if rt == nil {
		return nil, ErrNilRuntimeForServer
	}

	s := &ProduceBlockServer{
		runtime: rt,
		mux:     http.NewServeMux(),
	}

	s.mux.HandleFunc("/engine/produceBlock", s.handleProduceBlock)

	return s, nil
}

func (s *ProduceBlockServer) Handler() http.Handler {
	if s == nil {
		return http.NewServeMux()
	}
	return s.mux
}

func (s *ProduceBlockServer) handleProduceBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeRuntimeJSON(w, http.StatusMethodNotAllowed, produceBlockResponse{
			Error: "method not allowed",
		})
		return
	}

	var req ProduceBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, produceBlockResponse{
			Error: err.Error(),
		})
		return
	}

	result, err := s.runtime.ProduceBlock(req)
	if err != nil {
		writeRuntimeJSON(w, http.StatusBadRequest, produceBlockResponse{
			Result: result,
			Error:  err.Error(),
		})
		return
	}

	writeRuntimeJSON(w, http.StatusOK, produceBlockResponse{
		Result: result,
	})
}

func writeRuntimeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}
