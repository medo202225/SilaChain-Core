package engineapiserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/engineapi"
)

var (
	ErrNilService = errors.New("engineapiserver: nil service")
)

type Service interface {
	NewPayload(payload engineapi.PayloadEnvelope) (engineapi.PayloadStatus, error)
	ForkchoiceUpdated(state engineapi.ForkchoiceState) (engineapi.ForkchoiceUpdatedResult, error)
	ForkchoiceUpdatedWithAttributes(state engineapi.ForkchoiceState, attrs *blockassembly.PayloadAttributes) (engineapi.ForkchoiceUpdatedWithAttributesResult, error)
	GetPayload(payloadID string) (engineapi.GetPayloadResult, error)
	GetPayloadMetadata(payloadID string) (engineapi.PayloadMetadata, error)
}

type Server struct {
	service Service
	mux     *http.ServeMux
}

type newPayloadRequest struct {
	Payload engineapi.PayloadEnvelope `json:"payload"`
}

type newPayloadResponse struct {
	Status engineapi.PayloadStatus `json:"status"`
	Error  string                  `json:"error,omitempty"`
}

type forkchoiceUpdatedRequest struct {
	State             engineapi.ForkchoiceState        `json:"state"`
	PayloadAttributes *blockassembly.PayloadAttributes `json:"payload_attributes,omitempty"`
}

type forkchoiceUpdatedResponse struct {
	Result engineapi.ForkchoiceUpdatedWithAttributesResult `json:"result"`
	Error  string                                          `json:"error,omitempty"`
}

type getPayloadRequest struct {
	PayloadID string `json:"payload_id"`
}

type getPayloadResponse struct {
	Result engineapi.GetPayloadResult `json:"result"`
	Error  string                     `json:"error,omitempty"`
}

type getPayloadMetadataRequest struct {
	PayloadID string `json:"payload_id"`
}

type getPayloadMetadataResponse struct {
	Result engineapi.PayloadMetadata `json:"result"`
	Error  string                    `json:"error,omitempty"`
}

func New(service Service) (*Server, error) {
	if service == nil {
		return nil, ErrNilService
	}

	s := &Server{
		service: service,
		mux:     http.NewServeMux(),
	}

	s.mux.HandleFunc("/engine/newPayload", s.handleNewPayload)
	s.mux.HandleFunc("/engine/forkchoiceUpdated", s.handleForkchoiceUpdated)
	s.mux.HandleFunc("/engine/getPayload", s.handleGetPayload)
	s.mux.HandleFunc("/engine/getPayloadMetadata", s.handleGetPayloadMetadata)

	return s, nil
}

func (s *Server) Handler() http.Handler {
	if s == nil {
		return http.NewServeMux()
	}
	return s.mux
}

func (s *Server) handleNewPayload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, newPayloadResponse{
			Error: "method not allowed",
		})
		return
	}

	var req newPayloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, newPayloadResponse{
			Error: err.Error(),
		})
		return
	}

	status, err := s.service.NewPayload(req.Payload)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, newPayloadResponse{
			Status: status,
			Error:  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, newPayloadResponse{
		Status: status,
	})
}

func (s *Server) handleForkchoiceUpdated(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, forkchoiceUpdatedResponse{
			Error: "method not allowed",
		})
		return
	}

	var req forkchoiceUpdatedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, forkchoiceUpdatedResponse{
			Error: err.Error(),
		})
		return
	}

	result, err := s.service.ForkchoiceUpdatedWithAttributes(req.State, req.PayloadAttributes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, forkchoiceUpdatedResponse{
			Result: result,
			Error:  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, forkchoiceUpdatedResponse{
		Result: result,
	})
}

func (s *Server) handleGetPayload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, getPayloadResponse{
			Error: "method not allowed",
		})
		return
	}

	var req getPayloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, getPayloadResponse{
			Error: err.Error(),
		})
		return
	}

	result, err := s.service.GetPayload(req.PayloadID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, getPayloadResponse{
			Result: result,
			Error:  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, getPayloadResponse{
		Result: result,
	})
}

func (s *Server) handleGetPayloadMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, getPayloadMetadataResponse{
			Error: "method not allowed",
		})
		return
	}

	var req getPayloadMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, getPayloadMetadataResponse{
			Error: err.Error(),
		})
		return
	}

	result, err := s.service.GetPayloadMetadata(req.PayloadID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, getPayloadMetadataResponse{
			Result: result,
			Error:  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, getPayloadMetadataResponse{
		Result: result,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}
