# Sila API Reference

## Canonical Public Execution RPC Base
- `http://<execution-host>:8090`

## Canonical Consensus Runtime Base
- `http://127.0.0.1:8552`

## Canonical Engine Auth RPC Base
- `http://127.0.0.1:8551/engine`

---

## Public Read Endpoints

### Health
- `GET /health`

Returns node health status.

### Chain Info
- `GET /chain/info`

Returns chain metadata and current height.

### Block By Height
- `GET /blocks/height?height=<number>`

Returns block by canonical height.

### Transaction Submit
- `POST /tx/send`

Broadcasts a signed transaction.

### Explorer Summary
- `GET /explorer/summary`

Returns explorer summary metrics.

### Explorer Network
- `GET /explorer/network`

Returns network metrics exposed for explorer.

### Explorer Transaction
- `GET /explorer/tx?hash=<tx_hash>`

Returns transaction explorer view.

### Explorer Address
- `GET /explorer/address?address=<sila_address>`

Returns address explorer view.

### Explorer Block
- `GET /explorer/block?hash=<block_hash>`

Returns block explorer view.

### Explorer Logs
- `GET /explorer/logs?...`

Returns explorer log results.

### Explorer Contract
- `GET /explorer/contract?address=<sila_address>`

Returns contract explorer view.

---

## Consensus Runtime Read Endpoints

### Receipt By Tx Hash
- `GET /chain/receipt?tx_hash=<tx_hash>`

### Receipts By Block Hash
- `GET /chain/receiptsByBlock?hash=<block_hash>`

### Receipts By Block Number
- `GET /chain/receiptsByBlockNumber?number=<block_number>`

### Transactions By Block Hash
- `GET /chain/block/txs?hash=<block_hash>`

### Transactions By Block Number
- `GET /chain/block/txsByNumber?number=<block_number>`

### Transaction Lookup
- `GET /chain/tx?hash=<tx_hash>`

### Logs By Tx Hash
- `GET /chain/logs?tx_hash=<tx_hash>`

### State Read
- `GET /state/account?address=<sila_address>`

### Runtime Health
- `GET /healthz`

---

## Authenticated Engine API
JWT-protected engine RPC is exposed at:

- `POST /engine`

Supported methods include:
- `engine_exchangeCapabilities`
- `engine_identity`
- `engine_forkchoiceUpdatedV1`
- `engine_getPayloadV1`
- `engine_newPayloadV1`

---

## Notes
- Public explorer/read endpoints are read-only.
- Engine API is not a public explorer surface.
- Consensus runtime endpoints are operational/introspection endpoints and should be exposed intentionally.
- Canonical mainnet paths live under `config/networks/mainnet`.