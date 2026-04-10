package rpc

import (
	"net/http"
	"strconv"
	"strings"

	"silachain/internal/chain"
)

type LogsQueryRequest struct {
	Address   string `json:"address"`
	Event     string `json:"event"`
	Topic0    string `json:"topic0"`
	Topic1    string `json:"topic1"`
	Topic2    string `json:"topic2"`
	Topic3    string `json:"topic3"`
	FromBlock uint64 `json:"from_block"`
	ToBlock   uint64 `json:"to_block"`
}

func LogsQueryHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fromBlock, _ := strconv.ParseUint(strings.TrimSpace(r.URL.Query().Get("from_block")), 10, 64)
			toBlock, _ := strconv.ParseUint(strings.TrimSpace(r.URL.Query().Get("to_block")), 10, 64)

			logs := blockchain.QueryLogs(chain.LogQuery{
				Address:   strings.TrimSpace(r.URL.Query().Get("address")),
				Event:     strings.TrimSpace(r.URL.Query().Get("event")),
				Topic0:    strings.TrimSpace(r.URL.Query().Get("topic0")),
				Topic1:    strings.TrimSpace(r.URL.Query().Get("topic1")),
				Topic2:    strings.TrimSpace(r.URL.Query().Get("topic2")),
				Topic3:    strings.TrimSpace(r.URL.Query().Get("topic3")),
				FromBlock: fromBlock,
				ToBlock:   toBlock,
			})

			writeJSON(w, http.StatusOK, map[string]any{
				"count": len(logs),
				"logs":  logs,
			})
			return
		}

		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req LogsQueryRequest
		if err := decodeStrictJSON(r, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		logs := blockchain.QueryLogs(chain.LogQuery{
			Address:   strings.TrimSpace(req.Address),
			Event:     strings.TrimSpace(req.Event),
			Topic0:    strings.TrimSpace(req.Topic0),
			Topic1:    strings.TrimSpace(req.Topic1),
			Topic2:    strings.TrimSpace(req.Topic2),
			Topic3:    strings.TrimSpace(req.Topic3),
			FromBlock: req.FromBlock,
			ToBlock:   req.ToBlock,
		})

		writeJSON(w, http.StatusOK, map[string]any{
			"count": len(logs),
			"logs":  logs,
		})
	}
}
