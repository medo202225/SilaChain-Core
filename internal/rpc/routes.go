package rpc

import (
	"os"
	"strings"
	"time"

	"silachain/internal/chain"
	consensus "silachain/internal/consensus"
)

func RegisterRoutes(server *Server, blockchain *chain.Blockchain, state *consensus.ReadState) {
	sendTxRateLimiter := NewRateLimiter(time.Minute, 20)
	faucetRateLimiter := NewRateLimiter(time.Minute, 5)
	mineRateLimiter := NewRateLimiter(time.Minute, 5)
	attestationRateLimiter := NewRateLimiter(time.Minute, 60)
	contractWriteRateLimiter := NewRateLimiter(time.Minute, 20)

	adminToken := strings.TrimSpace(os.Getenv("SILA_ADMIN_TOKEN"))

	server.Router().HandleFunc("/health", secureRead(HealthHandler))
	server.Router().HandleFunc("/sila-rpc", secureLimitedJSONPost(sendTxRateLimiter, SilaRPCHandler(blockchain), MediumJSONBodyLimit))
	server.Router().HandleFunc("/chain/info", secureRead(ChainInfoHandler(blockchain)))
	server.Router().HandleFunc("/node/health", secureRead(NodeHealthHandler(blockchain)))
	server.Router().HandleFunc("/blocks/height", secureRead(BlockByHeightHandler(blockchain)))
	server.Router().HandleFunc("/tx/by-hash", secureRead(TxByHashHandler(blockchain)))
	server.Router().HandleFunc("/tx/receipt", secureRead(TxReceiptHandler(blockchain)))
	server.Router().HandleFunc("/sync/status", secureRead(SyncStatusHandler(blockchain)))
	server.Router().HandleFunc("/mempool/status", secureRead(MempoolStatusHandler(blockchain)))
	server.Router().HandleFunc("/logs/query", secureRead(LogsQueryHandler(blockchain)))

	server.Router().HandleFunc("/explorer", secureRead(ExplorerHomePageHandler()))
	server.Router().HandleFunc("/explorer/summary", secureRead(ExplorerSummaryHandler(blockchain)))
	server.Router().HandleFunc("/explorer/network", secureRead(NetworkStatusHandler(blockchain)))
	server.Router().HandleFunc("/explorer/contract", secureRead(ExplorerContractHandler(blockchain)))
	server.Router().HandleFunc("/explorer/tx-vm", secureRead(ExplorerTxVMHandler(blockchain)))
	server.Router().HandleFunc("/explorer/logs", secureRead(ExplorerLogsHandler(blockchain)))

	server.Router().HandleFunc("/consensus/attestations", secureRead(ListAttestationsHandler(state)))
	server.Router().HandleFunc("/consensus/attestations/aggregates", secureRead(AttestationAggregatesHandler(state, len(blockchain.ActiveValidators()))))
	server.Router().HandleFunc("/consensus/attestations/submit", secureLimitedJSONPost(attestationRateLimiter, SubmitAttestationHandler(state), StandardBodyLimitBytes))
	server.Router().HandleFunc("/consensus/justified", secureRead(JustifiedVotesHandler(state)))
	server.Router().HandleFunc("/consensus/finalized", secureRead(FinalizedVotesHandler(state)))

	server.Router().HandleFunc("/tx/send", secureLimitedJSONPost(sendTxRateLimiter, SendTxHandler(blockchain), MediumJSONBodyLimit))
	server.Router().HandleFunc("/contract/storage/get", secureRead(GetContractStorageHandler(blockchain)))

	if adminToken == "" {
		server.Router().HandleFunc("/contract/call", secureLocalOnlyJSONPost(contractWriteRateLimiter, ContractCallHandler(blockchain), MediumJSONBodyLimit))
		server.Router().HandleFunc("/contract/deploy", secureLocalOnlyJSONPost(contractWriteRateLimiter, DeployContractHandler(blockchain), MediumJSONBodyLimit))
		server.Router().HandleFunc("/faucet", secureLocalOnlyJSONPost(faucetRateLimiter, FaucetHandler(blockchain), SmallJSONBodyLimit))
		server.Router().HandleFunc("/mine", secureLocalOnlyJSONPost(mineRateLimiter, MineHandler(blockchain), SmallJSONBodyLimit))
		server.Router().HandleFunc("/contract/create", secureLocalOnlyJSONPost(contractWriteRateLimiter, CreateContractHandler(blockchain), MediumJSONBodyLimit))
		server.Router().HandleFunc("/contract/storage/set", secureLocalOnlyJSONPost(contractWriteRateLimiter, SetContractStorageHandler(blockchain), MediumJSONBodyLimit))
	} else {
		server.Router().HandleFunc("/contract/call", secureAdminJSONPost(adminToken, contractWriteRateLimiter, ContractCallHandler(blockchain), MediumJSONBodyLimit))
		server.Router().HandleFunc("/contract/deploy", secureAdminJSONPost(adminToken, contractWriteRateLimiter, DeployContractHandler(blockchain), MediumJSONBodyLimit))
		server.Router().HandleFunc("/faucet", secureAdminJSONPost(adminToken, faucetRateLimiter, FaucetHandler(blockchain), SmallJSONBodyLimit))
		server.Router().HandleFunc("/mine", secureAdminJSONPost(adminToken, mineRateLimiter, MineHandler(blockchain), SmallJSONBodyLimit))
		server.Router().HandleFunc("/contract/create", secureAdminJSONPost(adminToken, contractWriteRateLimiter, CreateContractHandler(blockchain), MediumJSONBodyLimit))
		server.Router().HandleFunc("/contract/storage/set", secureAdminJSONPost(adminToken, contractWriteRateLimiter, SetContractStorageHandler(blockchain), MediumJSONBodyLimit))
	}
}
