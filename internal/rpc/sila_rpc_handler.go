package rpc

import (
	"net/http"
	"strconv"
	"strings"

	"silachain/internal/chain"
	"silachain/internal/protocol"
	"silachain/pkg/types"
)

type SilaRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type SilaRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type SilaRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *SilaRPCError `json:"error,omitempty"`
}

type SilaCallParams struct {
	To        string `json:"to"`
	Input     string `json:"input"`
	VMVersion uint16 `json:"vm_version"`
	GasLimit  uint64 `json:"gas_limit"`
}

func silaHexUint64(v uint64) string {
	return "0x" + strconv.FormatUint(v, 16)
}

func silaHexAmount(v types.Amount) string {
	return "0x" + strconv.FormatUint(uint64(v), 16)
}

func parseSilaRPCAddress(param any) types.Address {
	if param == nil {
		return ""
	}

	raw, ok := param.(string)
	if !ok {
		return ""
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	return types.Address(raw)
}

func writeSilaRPC(w http.ResponseWriter, status int, resp SilaRPCResponse) {
	writeJSON(w, status, resp)
}

func SilaRPCHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req SilaRPCRequest
		if err := decodeStrictJSON(r, &req); err != nil {
			writeSilaRPC(w, http.StatusBadRequest, SilaRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &SilaRPCError{
					Code:    -32700,
					Message: err.Error(),
				},
			})
			return
		}

		if req.JSONRPC == "" {
			req.JSONRPC = "2.0"
		}

		switch req.Method {
		case "sila_chainId":
			writeSilaRPC(w, http.StatusOK, SilaRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  silaHexUint64(uint64(protocol.DefaultMainnetParams().ChainID)),
			})
			return

		case "sila_blockNumber":
			height, err := blockchain.Height()
			if err != nil {
				writeSilaRPC(w, http.StatusInternalServerError, SilaRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &SilaRPCError{
						Code:    -32000,
						Message: err.Error(),
					},
				})
				return
			}

			writeSilaRPC(w, http.StatusOK, SilaRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  silaHexUint64(uint64(height)),
			})
			return

		case "sila_getBalance":
			if len(req.Params) < 1 {
				writeSilaRPC(w, http.StatusBadRequest, SilaRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &SilaRPCError{
						Code:    -32602,
						Message: "missing address param",
					},
				})
				return
			}

			address := parseSilaRPCAddress(req.Params[0])
			if address == "" {
				writeSilaRPC(w, http.StatusBadRequest, SilaRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &SilaRPCError{
						Code:    -32602,
						Message: "invalid address param",
					},
				})
				return
			}

			acc, err := blockchain.GetAccount(address)
			if err != nil {
				writeSilaRPC(w, http.StatusOK, SilaRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result:  silaHexAmount(0),
				})
				return
			}

			writeSilaRPC(w, http.StatusOK, SilaRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  silaHexAmount(acc.Balance),
			})
			return

		case "sila_call":
			if len(req.Params) < 1 {
				writeSilaRPC(w, http.StatusBadRequest, SilaRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &SilaRPCError{
						Code:    -32602,
						Message: "missing call params",
					},
				})
				return
			}

			raw, ok := req.Params[0].(map[string]any)
			if !ok {
				writeSilaRPC(w, http.StatusBadRequest, SilaRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &SilaRPCError{
						Code:    -32602,
						Message: "invalid call params object",
					},
				})
				return
			}

			to, _ := raw["to"].(string)
			input, _ := raw["input"].(string)

			var vmVersion uint16 = 1
			if v, ok := raw["vm_version"].(float64); ok && v > 0 {
				vmVersion = uint16(v)
			}

			var gasLimit uint64 = 100000
			if v, ok := raw["gas_limit"].(float64); ok && v > 0 {
				gasLimit = uint64(v)
			}

			result, err := blockchain.ReadOnlyCall(types.Address(strings.TrimSpace(to)), strings.TrimSpace(input), vmVersion, gasLimit)
			if err != nil {
				writeSilaRPC(w, http.StatusBadRequest, SilaRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &SilaRPCError{
						Code:    -32000,
						Message: err.Error(),
					},
				})
				return
			}

			writeSilaRPC(w, http.StatusOK, SilaRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  result,
			})
			return

		default:
			writeSilaRPC(w, http.StatusBadRequest, SilaRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &SilaRPCError{
					Code:    -32601,
					Message: "method not found",
				},
			})
			return
		}
	}
}
