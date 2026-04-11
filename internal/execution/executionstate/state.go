package executionstate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	corestate "silachain/core/state"
	"sort"
	"sync"
)

const (
	IntrinsicGasBase     uint64 = 21000
	IntrinsicGasPerByte  uint64 = 16
	DefaultTxFeePerGas   uint64 = 1
	DefaultBlockGasLimit uint64 = 30000000
)

type ImportedBlock struct {
	Number     uint64
	Hash       string
	ParentHash string
	Timestamp  uint64
	TxHashes   []string
}

type PendingTx struct {
	Hash     string
	From     string
	To       string
	Value    uint64
	Nonce    uint64
	Data     string
	Fee      uint64
	GasLimit uint64
}

type Receipt struct {
	TxHash          string
	BlockNumber     uint64
	BlockHash       string
	From            string
	To              string
	GasUsed         uint64
	EffectiveGasFee uint64
	Success         bool
}

type BlockExecutionRequest struct {
	Block ImportedBlock
	Txs   []PendingTx
}

type BlockExecutionResult struct {
	BlockHash   string
	BlockNumber uint64
	StateRoot   string
	GasUsed     uint64
	Receipts    []Receipt
}

type State struct {
	mu             sync.RWMutex
	genesisHash    string
	headNumber     uint64
	headHash       string
	blocks         map[uint64]ImportedBlock
	pendingTxs     map[string]PendingTx
	db             *corestate.StateDB
	receiptsByHash map[string]Receipt
	lastBlockGas   uint64
}

func NewState(genesisHash string) *State {
	s := &State{
		genesisHash:    genesisHash,
		headNumber:     0,
		headHash:       genesisHash,
		blocks:         make(map[uint64]ImportedBlock),
		pendingTxs:     make(map[string]PendingTx),
		db:             corestate.NewStateDB(),
		receiptsByHash: make(map[string]Receipt),
	}

	s.blocks[0] = ImportedBlock{
		Number:     0,
		Hash:       genesisHash,
		ParentHash: "",
		Timestamp:  0,
		TxHashes:   nil,
	}
	return s
}

func ValidateBlock(currentHeadHash string, currentHeadNumber uint64, block ImportedBlock) error {
	if block.Hash == "" {
		return fmt.Errorf("execution state: empty block hash")
	}
	if block.Number == 0 {
		return fmt.Errorf("execution state: genesis re-import is not allowed")
	}
	if block.ParentHash == "" {
		return fmt.Errorf("execution state: empty parent hash")
	}
	if block.Number != currentHeadNumber+1 {
		return fmt.Errorf("execution state: non-sequential block number %d", block.Number)
	}
	if block.ParentHash != currentHeadHash {
		return fmt.Errorf("execution state: parent hash mismatch for block %d", block.Number)
	}
	for i, txHash := range block.TxHashes {
		if txHash == "" {
			return fmt.Errorf("execution state: empty tx hash at index %d", i)
		}
	}
	return nil
}

func ValidateTx(tx PendingTx) error {
	if tx.Hash == "" {
		return fmt.Errorf("execution state: empty tx hash")
	}
	if tx.From == "" {
		return fmt.Errorf("execution state: empty from")
	}
	if tx.To == "" {
		return fmt.Errorf("execution state: empty to")
	}
	return nil
}

func IntrinsicGas(tx PendingTx) uint64 {
	return IntrinsicGasBase + uint64(len(tx.Data))*IntrinsicGasPerByte
}

func NormalizeTx(tx PendingTx) PendingTx {
	if tx.Fee == 0 {
		tx.Fee = DefaultTxFeePerGas
	}
	if tx.GasLimit == 0 {
		tx.GasLimit = IntrinsicGas(tx)
	}
	return tx
}

func (s *State) ensureAccount(address string) *corestate.Account {
	if s.db == nil {
		return nil
	}
	return s.db.EnsureAccount(address)
}

func (s *State) SetBalance(address string, balance uint64) {
	if s.db == nil {
		return
	}
	s.db.SetBalance(address, balance)
}

func (s *State) GetBalance(address string) uint64 {
	if s.db == nil {
		return 0
	}
	return s.db.GetBalance(address)
}

func (s *State) GetNonce(address string) uint64 {
	if s.db == nil {
		return 0
	}
	return s.db.GetNonce(address)
}

func (s *State) AddPendingTx(tx PendingTx) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx = NormalizeTx(tx)

	if tx.Hash == "" {
		return false
	}
	if _, ok := s.pendingTxs[tx.Hash]; ok {
		return false
	}
	s.pendingTxs[tx.Hash] = tx
	return true
}

func (s *State) HasPendingTx(hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.pendingTxs[hash]
	return ok
}

func (s *State) PendingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pendingTxs)
}

func (s *State) ApplyTransaction(tx PendingTx) error {
	_, err := s.ApplyTransactionInBlock(tx, 0, "")
	return err
}

func (s *State) ApplyTransactionInBlock(tx PendingTx, blockNumber uint64, blockHash string) (Receipt, error) {
	if err := ValidateTx(tx); err != nil {
		return Receipt{}, err
	}

	tx = NormalizeTx(tx)
	gasUsed := IntrinsicGas(tx)

	if tx.GasLimit < gasUsed {
		return Receipt{}, fmt.Errorf("execution state: gas limit below intrinsic gas")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return Receipt{}, fmt.Errorf("execution state: nil state db")
	}

	fromNonce := s.db.GetNonce(tx.From)
	if fromNonce != tx.Nonce {
		return Receipt{}, fmt.Errorf("execution state: invalid nonce for %s", tx.From)
	}

	effectiveGasFee := gasUsed * tx.Fee
	totalCost := tx.Value + effectiveGasFee

	fromBalance := s.db.GetBalance(tx.From)
	if fromBalance < totalCost {
		return Receipt{}, fmt.Errorf("execution state: insufficient balance for %s", tx.From)
	}

	toBalance := s.db.GetBalance(tx.To)

	s.db.SetBalance(tx.From, fromBalance-totalCost)
	s.db.SetNonce(tx.From, fromNonce+1)
	s.db.SetBalance(tx.To, toBalance+tx.Value)

	delete(s.pendingTxs, tx.Hash)

	receipt := Receipt{
		TxHash:          tx.Hash,
		BlockNumber:     blockNumber,
		BlockHash:       blockHash,
		From:            tx.From,
		To:              tx.To,
		GasUsed:         gasUsed,
		EffectiveGasFee: effectiveGasFee,
		Success:         true,
	}

	s.receiptsByHash[tx.Hash] = receipt
	return receipt, nil
}

func (s *State) ExecuteBlock(req BlockExecutionRequest) (BlockExecutionResult, error) {
	if err := ValidateBlock(s.HeadHash(), s.HeadNumber(), req.Block); err != nil {
		return BlockExecutionResult{}, err
	}
	if len(req.Block.TxHashes) != len(req.Txs) {
		return BlockExecutionResult{}, fmt.Errorf("execution state: tx count mismatch for block")
	}

	var totalGasUsed uint64
	receipts := make([]Receipt, 0, len(req.Txs))

	for i, tx := range req.Txs {
		if req.Block.TxHashes[i] != tx.Hash {
			return BlockExecutionResult{}, fmt.Errorf("execution state: tx hash mismatch at index %d", i)
		}

		tx = NormalizeTx(tx)
		gasUsed := IntrinsicGas(tx)
		if totalGasUsed+gasUsed > DefaultBlockGasLimit {
			return BlockExecutionResult{}, fmt.Errorf("execution state: block gas limit exceeded")
		}

		receipt, err := s.ApplyTransactionInBlock(tx, req.Block.Number, req.Block.Hash)
		if err != nil {
			return BlockExecutionResult{}, err
		}

		totalGasUsed += receipt.GasUsed
		receipts = append(receipts, receipt)
	}

	if err := s.ImportBlock(req.Block); err != nil {
		return BlockExecutionResult{}, err
	}

	s.mu.Lock()
	s.lastBlockGas = totalGasUsed
	stateRoot := s.computeStateRootLocked()
	s.mu.Unlock()

	return BlockExecutionResult{
		BlockHash:   req.Block.Hash,
		BlockNumber: req.Block.Number,
		StateRoot:   stateRoot,
		GasUsed:     totalGasUsed,
		Receipts:    receipts,
	}, nil
}

func (s *State) FinalizeBlockExecution(block ImportedBlock, totalGasUsed uint64) (string, error) {
	if err := s.ImportBlock(block); err != nil {
		return "", err
	}

	s.mu.Lock()
	s.lastBlockGas = totalGasUsed
	stateRoot := s.computeStateRootLocked()
	s.mu.Unlock()

	return stateRoot, nil
}

func (s *State) computeStateRootLocked() string {
	snapshot := map[string]corestate.Account{}
	if s.db != nil {
		snapshot = s.db.SnapshotAccounts()
	}

	addresses := make([]string, 0, len(snapshot))
	for address := range snapshot {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)

	h := sha256.New()
	for _, address := range addresses {
		account := snapshot[address]
		_, _ = h.Write([]byte(fmt.Sprintf("%s|%d|%d;", address, account.Balance, account.Nonce)))
	}

	return "0x" + hex.EncodeToString(h.Sum(nil))
}

func (s *State) ImportBlock(block ImportedBlock) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.blocks[block.Number]; ok {
		if existing.Hash == block.Hash {
			return nil
		}
		return fmt.Errorf("execution state: conflicting block at height %d", block.Number)
	}

	if err := ValidateBlock(s.headHash, s.headNumber, block); err != nil {
		return err
	}

	s.blocks[block.Number] = block
	s.headNumber = block.Number
	s.headHash = block.Hash
	return nil
}

func (s *State) GetReceipt(txHash string) (Receipt, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	receipt, ok := s.receiptsByHash[txHash]
	return receipt, ok
}

func (s *State) LastBlockGasUsed() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastBlockGas
}

func (s *State) HeadNumber() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.headNumber
}

func (s *State) HeadHash() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.headHash
}

func (s *State) AccountNonce(address string) uint64 {
	if s.db == nil {
		return 0
	}
	return s.db.AccountNonce(address)
}
