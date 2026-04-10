package txpoolapi

import (
	"encoding/json"
	"errors"
	"net/http"
)

var (
	ErrNilService = errors.New("txpoolapi: nil service")
)

type API interface {
	Add(req AddTxRequest) (AddTxResult, error)
	Status() (StatusResult, error)
}

type Server struct {
	api API
	mux *http.ServeMux
}

type addResponse struct {
	Result AddTxResult `json:"result"`
	Error  string      `json:"error,omitempty"`
}

type statusResponse struct {
	Result StatusResult `json:"result"`
	Error  string       `json:"error,omitempty"`
}

func NewServer(api API) (*Server, error) {
	if api == nil {
		return nil, ErrNilService
	}

	s := &Server{
		api: api,
		mux: http.NewServeMux(),
	}

	s.mux.HandleFunc("/txpool/add", s.handleAdd)
	s.mux.HandleFunc("/txpool/status", s.handleStatus)

	return s, nil
}

func (s *Server) Handler() http.Handler {
	if s == nil {
		return http.NewServeMux()
	}
	return s.mux
}

func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, addResponse{
			Error: "method not allowed",
		})
		return
	}

	var req AddTxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, addResponse{
			Error: err.Error(),
		})
		return
	}

	result, err := s.api.Add(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, addResponse{
			Result: result,
			Error:  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, addResponse{
		Result: result,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, statusResponse{
			Error: "method not allowed",
		})
		return
	}

	result, err := s.api.Status()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, statusResponse{
			Result: result,
			Error:  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, statusResponse{
		Result: result,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}
