package rpc

import (
	"encoding/json"
	"net/http"
	"strings"

	"silachain/internal/chain"
	"silachain/pkg/types"
)

func ExplorerSummaryHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		height, _ := blockchain.Height()
		latest, _ := blockchain.LatestBlock()

		validators := blockchain.Validators()
		active := blockchain.ActiveValidators()
		jails := blockchain.Jails()
		stakes := blockchain.Stakes()
		delegations := blockchain.Delegations()
		metrics := blockchain.MonetaryMetrics()

		var totalSelfStake uint64
		for _, s := range stakes {
			totalSelfStake += s.Stake
		}

		var totalDelegated uint64
		for _, d := range delegations {
			totalDelegated += d.Amount
		}

		jailedNow := 0
		for _, j := range jails {
			if j.Jailed {
				jailedNow++
			}
		}

		latestHash := ""
		latestHeight := uint64(0)
		latestProposer := ""
		latestTxCount := 0
		latestTimestamp := int64(0)

		if latest != nil {
			latestHash = string(latest.Header.Hash)
			latestHeight = uint64(latest.Header.Height)
			latestProposer = string(latest.Header.Proposer)
			latestTxCount = len(latest.Transactions)
			latestTimestamp = int64(latest.Header.Timestamp)
		}

		out := map[string]any{
			"chain": map[string]any{
				"height":      height,
				"latest_hash": latestHash,
			},
			"latest_block": map[string]any{
				"height":    latestHeight,
				"hash":      latestHash,
				"proposer":  latestProposer,
				"tx_count":  latestTxCount,
				"timestamp": latestTimestamp,
			},
			"mempool": map[string]any{
				"count": blockchain.Mempool().Count(),
			},
			"validators": map[string]any{
				"total":        len(validators),
				"active":       len(active),
				"jailed_now":   jailedNow,
				"jail_records": len(jails),
			},
			"staking": map[string]any{
				"self_stake_total":      totalSelfStake,
				"delegated_stake_total": totalDelegated,
			},
			"monetary_metrics": metrics,
			"monetary_policy": map[string]any{
				"burn_enabled":             false,
				"treasury_enabled":         false,
				"monetary_policy_frozen":   true,
				"block_reward":             10,
				"unbonding_delay":          3,
				"min_validator_stake":      1,
				"validator_commission_bps": 1000,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func ValidatorDetailHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := types.Address(r.URL.Query().Get("address"))

		var selfStake uint64
		for _, s := range blockchain.Stakes() {
			if s.Validator == address {
				selfStake = s.Stake
				break
			}
		}

		var delegatedStake uint64
		for _, d := range blockchain.Delegations() {
			if d.Validator == address {
				delegatedStake += d.Amount
			}
		}

		jailed := false
		for _, j := range blockchain.Jails() {
			if j.Validator == address && j.Jailed {
				jailed = true
				break
			}
		}

		active := false
		for _, v := range blockchain.ActiveValidators() {
			if v.Address == address {
				active = true
				break
			}
		}

		out := map[string]any{
			"address":         address,
			"self_stake":      selfStake,
			"delegated_stake": delegatedStake,
			"total_stake":     selfStake + delegatedStake,
			"jailed":          jailed,
			"active":          active,
			"pending_rewards": blockchain.PendingRewards(address),
			"pending_unbond":  blockchain.PendingUnbond(address),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func NetworkStatusHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		height, _ := blockchain.Height()
		latest, _ := blockchain.LatestBlock()

		latestHash := ""
		if latest != nil {
			latestHash = string(latest.Header.Hash)
		}

		out := map[string]any{
			"status":            "ok",
			"height":            height,
			"latest_hash":       latestHash,
			"mempool_count":     blockchain.Mempool().Count(),
			"validators_total":  len(blockchain.Validators()),
			"validators_active": len(blockchain.ActiveValidators()),
			"monetary_metrics":  blockchain.MonetaryMetrics(),
			"monetary_policy": map[string]any{
				"burn_enabled":             false,
				"treasury_enabled":         false,
				"monetary_policy_frozen":   true,
				"block_reward":             10,
				"unbonding_delay":          3,
				"min_validator_stake":      1,
				"validator_commission_bps": 1000,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func ExplorerContractHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := types.Address(strings.TrimSpace(r.URL.Query().Get("address")))
		if address == "" {
			writeJSONError(w, http.StatusBadRequest, "missing address")
			return
		}

		acc, err := blockchain.GetAccount(address)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}

		code, ok := blockchain.GetContractCode(address)
		storageRoot, _ := blockchain.GetContractStorageRoot(address)

		writeJSON(w, http.StatusOK, map[string]any{
			"address":      address,
			"is_contract":  acc.IsContract(),
			"balance":      acc.Balance,
			"nonce":        acc.Nonce,
			"code_hash":    acc.CodeHash,
			"storage_root": storageRoot,
			"code_found":   ok,
			"code":         code,
			"code_size":    len(code),
		})
	}
}

func ExplorerTxVMHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := strings.TrimSpace(r.URL.Query().Get("hash"))
		if hash == "" {
			writeJSONError(w, http.StatusBadRequest, "missing hash")
			return
		}

		txObj, height, ok := blockchain.GetTransactionByHash(hash)
		if !ok {
			writeJSONError(w, http.StatusNotFound, "transaction not found")
			return
		}

		receipt, ok := blockchain.GetReceiptByHash(hash)
		if !ok {
			writeJSONError(w, http.StatusNotFound, "receipt not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"height":      height,
			"transaction": txObj,
			"receipt":     receipt,
			"vm": map[string]any{
				"success":         receipt.Success,
				"gas_used":        receipt.GasUsed,
				"return_data":     receipt.ReturnData,
				"revert_data":     receipt.RevertData,
				"created_address": receipt.CreatedAddress,
				"logs_count":      len(receipt.Logs),
			},
		})
	}
}

func ExplorerLogsHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := strings.TrimSpace(r.URL.Query().Get("address"))
		event := strings.TrimSpace(r.URL.Query().Get("event"))
		topic0 := strings.TrimSpace(r.URL.Query().Get("topic0"))
		topic1 := strings.TrimSpace(r.URL.Query().Get("topic1"))
		topic2 := strings.TrimSpace(r.URL.Query().Get("topic2"))
		topic3 := strings.TrimSpace(r.URL.Query().Get("topic3"))

		logs := blockchain.QueryLogs(chain.LogQuery{
			Address: address,
			Event:   event,
			Topic0:  topic0,
			Topic1:  topic1,
			Topic2:  topic2,
			Topic3:  topic3,
		})

		writeJSON(w, http.StatusOK, map[string]any{
			"count": logsCountExplorer(logs),
			"logs":  logs,
		})
	}
}

func logsCountExplorer(logs []chain.LogRecord) int {
	return len(logs)
}
