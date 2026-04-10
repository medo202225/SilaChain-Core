package execution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type EngineServer struct {
	jwtSecret       string
	listenAddress   string
	executionRPCURL string
	httpServer      *http.Server

	mu       sync.Mutex
	payloads map[string]ExecutionPayloadV1
}

func NewEngineServer(listenAddress string, jwtSecret string, executionRPCURL string) *EngineServer {
	s := &EngineServer{
		jwtSecret:       jwtSecret,
		listenAddress:   listenAddress,
		executionRPCURL: executionRPCURL,
		payloads:        make(map[string]ExecutionPayloadV1),
	}
	s.httpServer = &http.Server{
		Addr:    listenAddress,
		Handler: s.Handler(),
	}
	return s
}

func (s *EngineServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/engine", EngineJWTMiddleware(s.jwtSecret, http.HandlerFunc(s.handleEngineRPC)))
	return mux
}

func (s *EngineServer) Start() error {
	if s == nil || s.httpServer == nil {
		return fmt.Errorf("engine server is nil")
	}
	return s.httpServer.ListenAndServe()
}

func payloadIDFromForkchoice(head string, timestamp string, prevRandao string) string {
	sum := sha256.Sum256([]byte(head + "|" + timestamp + "|" + prevRandao))
	return "0x" + hex.EncodeToString(sum[:8])
}

func (s *EngineServer) handleEngineRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	method, _ := req["method"].(string)
	id := req["id"]

	out := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
	}

	switch method {
	case "engine_exchangeCapabilities":
		out["result"] = []string{
			"engine_exchangeCapabilities",
			"engine_identity",
			"engine_forkchoiceUpdatedV1",
			"engine_getPayloadV1",
			"engine_newPayloadV1",
		}

	case "engine_identity":
		out["result"] = map[string]any{
			"listen_address": s.listenAddress,
			"execution_rpc":  s.executionRPCURL,
			"auth":           "jwt",
		}

	case "engine_forkchoiceUpdatedV1":
		params, _ := req["params"].([]any)
		var fc ForkchoiceStateV1
		var attrs *PayloadAttributesV1

		if len(params) >= 1 {
			raw, _ := json.Marshal(params[0])
			_ = json.Unmarshal(raw, &fc)
		}
		if len(params) >= 2 && params[1] != nil {
			raw, _ := json.Marshal(params[1])
			var parsed PayloadAttributesV1
			if err := json.Unmarshal(raw, &parsed); err == nil {
				attrs = &parsed
			}
		}

		latestValidHash := fc.HeadBlockHash
		result := ForkchoiceUpdatedResponseV1{
			PayloadStatus: PayloadStatusV1{
				Status:          "VALID",
				LatestValidHash: &latestValidHash,
				ValidationError: nil,
			},
			PayloadID: nil,
		}

		if attrs != nil {
			payloadID := payloadIDFromForkchoice(fc.HeadBlockHash, attrs.Timestamp, attrs.PrevRandao)
			result.PayloadID = &payloadID

			payload := ExecutionPayloadV1{
				BlockHash:    fc.HeadBlockHash,
				ParentHash:   fc.SafeBlockHash,
				BlockNumber:  attrs.Timestamp,
				Timestamp:    attrs.Timestamp,
				PrevRandao:   attrs.PrevRandao,
				FeeRecipient: attrs.SuggestedFeeRecipient,
			}

			s.mu.Lock()
			s.payloads[payloadID] = payload
			s.mu.Unlock()
		}

		out["result"] = result

	case "engine_getPayloadV1":
		params, _ := req["params"].([]any)
		if len(params) < 1 {
			out["error"] = map[string]any{
				"code":    -32602,
				"message": "missing payload id",
			}
			break
		}

		payloadID, _ := params[0].(string)

		s.mu.Lock()
		payload, ok := s.payloads[payloadID]
		s.mu.Unlock()

		if !ok {
			out["error"] = map[string]any{
				"code":    -32001,
				"message": "unknown payload id",
			}
			break
		}

		out["result"] = GetPayloadResponseV1{
			ExecutionPayload: payload,
			BlockValue:       "0x0",
		}

	case "engine_newPayloadV1":
		params, _ := req["params"].([]any)
		if len(params) < 1 {
			out["error"] = map[string]any{
				"code":    -32602,
				"message": "missing execution payload",
			}
			break
		}

		var payload ExecutionPayloadV1
		raw, _ := json.Marshal(params[0])
		_ = json.Unmarshal(raw, &payload)

		latestValidHash := payload.BlockHash
		out["result"] = PayloadStatusV1{
			Status:          "VALID",
			LatestValidHash: &latestValidHash,
			ValidationError: nil,
		}

	default:
		out["error"] = map[string]any{
			"code":    -32601,
			"message": "method not found",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
