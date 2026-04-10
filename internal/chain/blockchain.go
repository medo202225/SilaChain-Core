package chain

import (
	"errors"

	"silachain/internal/accounts"
	block "silachain/internal/block"
	"silachain/internal/config"
	"silachain/internal/core/state"
	coretypes "silachain/internal/core/types"
	chainGenesis "silachain/internal/genesis"
	"silachain/internal/mempool"
	"silachain/internal/staking"
	"silachain/internal/storage"
	"silachain/internal/validator"
	pkgtypes "silachain/pkg/types"
)

var (
	ErrNilBlockchain = errors.New("blockchain is nil")
	ErrEmptyChain    = errors.New("chain is empty")
	ErrInvalidParent = errors.New("invalid parent hash")
	ErrInvalidHeight = errors.New("invalid block height")
	ErrEmptyMempool  = errors.New("mempool is empty")
)

type Blockchain struct {
	blocks                 []*coretypes.Block
	accounts               *accounts.Manager
	state                  *state.Manager
	transition             *state.Transition
	executor               *state.Executor
	contractStorage        *state.ContractStorage
	contractRegistry       *state.ContractRegistry
	stateCommitment        state.StateCommitment
	mempool                *mempool.Pool
	receipts               map[string]coretypes.Receipt
	txIndex                map[string]storage.TxLocation
	validatorSet           *validator.Set
	validatorReg           *validator.Registry
	stakingReg             *staking.Registry
	delegationReg          *staking.DelegationRegistry
	undelegationReg        *staking.UndelegationRegistry
	slashReg               *staking.SlashRegistry
	rewardReg              *staking.RewardRegistry
	delegatorRewardReg     *staking.DelegatorRewardRegistry
	withdrawalReg          *staking.WithdrawalRegistry
	jailReg                *staking.JailRegistry
	unbondClaimReg         *staking.UnbondClaimRegistry
	activeValidators       []validator.Member
	weightedValidators     []validator.Member
	minValidatorStake      uint64
	blockReward            uint64
	unbondingDelay         uint64
	validatorCommissionBps uint64
	meta                   storage.ChainMetadata
	db                     *storage.DB
	blockStore             *storage.BlockStore
	accountStore           *storage.AccountStore
	receiptStore           *storage.ReceiptStore
	txStore                *storage.TxStore
	metaStore              *storage.MetadataStore
	contractCodeStore      *storage.ContractCodeStore
	contractStorageStore   *storage.ContractStorageStore
	validatorStore         *storage.ValidatorStore
	stakingStore           *storage.StakingStore
	delegationStore        *storage.DelegationStore
	undelegationStore      *storage.UndelegationStore
	slashStore             *storage.SlashStore
	rewardStore            *storage.RewardStore
	delegatorRewardStore   *storage.DelegatorRewardStore
	withdrawalStore        *storage.WithdrawalStore
	jailStore              *storage.JailStore
	unbondClaimStore       *storage.UnbondClaimStore
	stateStore             *storage.StateStore
}

func NewBlockchain(dataDir string, validatorSet *validator.Set, minValidatorStake uint64, opts ...Option) (*Blockchain, error) {
	if dataDir == "" {
		dataDir = "data/node"
	}

	protocolCfg, err := config.LoadProtocolConfig("config/networks/mainnet/public/protocol.json")
	if err != nil {
		return nil, err
	}

	if minValidatorStake == 0 {
		minValidatorStake = protocolCfg.MinValidatorStake
	}

	options := defaultBlockchainOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.stateCommitment == nil {
		options.stateCommitment = state.NewHashStateCommitment()
	}

	accountMgr := accounts.NewManager()
	stateMgr := state.NewManager(accountMgr)
	contractStorage := state.NewContractStorage()
	contractRegistry := state.NewContractRegistry(stateMgr, contractStorage)
	stateCommitment := options.stateCommitment
	transition := state.NewTransition(stateMgr, contractRegistry)
	transition.SetVMExecutor(
		newTransitionVMExecutor(
			transition.CodeRegistry(),
			transition.Storage(),
			transition.Journal(),
		),
	)
	executor := state.NewExecutor(transition)
	pool := mempool.NewPool()
	registry := validator.NewRegistry()
	stakingReg := staking.NewRegistry()
	delegationReg := staking.NewDelegationRegistry()
	undelegationReg := staking.NewUndelegationRegistry()
	slashReg := staking.NewSlashRegistry()
	rewardReg := staking.NewRewardRegistry()
	delegatorRewardReg := staking.NewDelegatorRewardRegistry()
	withdrawalReg := staking.NewWithdrawalRegistry()
	jailReg := staking.NewJailRegistry()
	unbondClaimReg := staking.NewUnbondClaimRegistry()

	db := storage.NewDB(dataDir)
	blockStore := storage.NewBlockStore(db)
	accountStore := storage.NewAccountStore(db)
	receiptStore := storage.NewReceiptStore(db)
	txStore := storage.NewTxStore(db)
	metaStore := storage.NewMetadataStore(db)
	contractCodeStore := storage.NewContractCodeStore(db)
	contractStorageStore := storage.NewContractStorageStore(db)
	stateStore := storage.NewStateStore(db)
	validatorStore := storage.NewValidatorStore(db)
	stakingStore := storage.NewStakingStore(db)
	delegationStore := storage.NewDelegationStore(db)
	undelegationStore := storage.NewUndelegationStore(db)
	slashStore := storage.NewSlashStore(db)
	rewardStore := storage.NewRewardStore(db)
	delegatorRewardStore := storage.NewDelegatorRewardStore(db)
	withdrawalStore := storage.NewWithdrawalStore(db)
	jailStore := storage.NewJailStore(db)
	unbondClaimStore := storage.NewUnbondClaimStore(db)

	loadedBlocks, err := blockStore.Load()
	if err != nil {
		return nil, err
	}

	loadedAccounts, err := accountStore.Load()
	if err != nil {
		return nil, err
	}

	loadedReceipts, err := receiptStore.Load()
	if err != nil {
		return nil, err
	}

	loadedTxIndex, err := txStore.Load()
	if err != nil {
		return nil, err
	}

	loadedMeta, err := metaStore.Load()
	if err != nil {
		return nil, err
	}

	loadedContractCode, err := contractCodeStore.Load()
	if err != nil {
		return nil, err
	}

	loadedContractStorage, err := contractStorageStore.Load()
	if err != nil {
		return nil, err
	}

	if trieCommitment, ok := stateCommitment.(*state.TrieStateCommitment); ok && trieCommitment != nil {
		trieCommitment.WithNodeStore(stateStore)
	}
	loadedValidators, err := validatorStore.Load()
	if err != nil {
		return nil, err
	}

	loadedStakes, err := stakingStore.Load()
	if err != nil {
		return nil, err
	}

	loadedDelegations, err := delegationStore.Load()
	if err != nil {
		return nil, err
	}

	loadedUndelegations, err := undelegationStore.Load()
	if err != nil {
		return nil, err
	}

	loadedSlashes, err := slashStore.Load()
	if err != nil {
		return nil, err
	}

	loadedRewards, err := rewardStore.Load()
	if err != nil {
		return nil, err
	}

	loadedDelegatorRewards, err := delegatorRewardStore.Load()
	if err != nil {
		return nil, err
	}

	loadedWithdrawals, err := withdrawalStore.Load()
	if err != nil {
		return nil, err
	}

	loadedJails, err := jailStore.Load()
	if err != nil {
		return nil, err
	}

	loadedUnbondClaims, err := unbondClaimStore.Load()
	if err != nil {
		return nil, err
	}

	if len(loadedBlocks) > 0 {
		if err := validateLoadedBlocks(loadedBlocks); err != nil {
			return nil, err
		}
	}

	if len(loadedBlocks) == 0 {
		genesisCfg, genesisErr := chainGenesis.Load("config/networks/mainnet/public/genesis.json")

		if genesisErr == nil {
			if err := chainGenesis.ApplyAccounts(accountMgr, genesisCfg); err != nil {
				return nil, err
			}
		}

		root, err := stateCommitment.ComputeStateRoot(accountMgr)
		if err != nil {
			return nil, err
		}

		genesisBlock, err := NewGenesisBlock()
		if err != nil {
			return nil, err
		}

		genesisBlock.Header.StateRoot = root

		txRoot, err := block.TxRootHash(genesisBlock.Transactions)
		if err != nil {
			return nil, err
		}
		genesisBlock.Header.TxRoot = txRoot

		receiptRoot, err := block.ReceiptRootHash(genesisBlock.Receipts)
		if err != nil {
			return nil, err
		}
		genesisBlock.Header.ReceiptRoot = receiptRoot

		hash, err := block.HeaderHash(genesisBlock.Header)
		if err != nil {
			return nil, err
		}
		genesisBlock.Header.Hash = hash

		loadedBlocks = []*coretypes.Block{genesisBlock}
	} else {
		for _, acc := range loadedAccounts {
			_ = accountMgr.Set(acc)
		}
	}

	if len(loadedContractCode) > 0 {
		contractRegistry.LoadCode(loadedContractCode)
	}
	if len(loadedContractStorage) > 0 {
		contractRegistry.LoadStorage(loadedContractStorage)
	}

	if len(loadedValidators) > 0 {
		registry.LoadFromSet(validator.NewSet(loadedValidators))
	} else if validatorSet != nil {
		registry.LoadFromSet(validatorSet)
	}

	if len(loadedStakes) > 0 {
		stakingReg.Load(loadedStakes)
	} else if validatorSet != nil {
		for _, m := range validatorSet.All() {
			stakingReg.Set(m.Address, m.Stake)
		}
	}

	if len(loadedDelegations) > 0 {
		delegationReg.Load(loadedDelegations)
	}
	if len(loadedUndelegations) > 0 {
		undelegationReg.Load(loadedUndelegations)
	}
	if len(loadedSlashes) > 0 {
		slashReg.Load(loadedSlashes)
	}
	if len(loadedRewards) > 0 {
		rewardReg.Load(loadedRewards)
	}
	if len(loadedDelegatorRewards) > 0 {
		delegatorRewardReg.Load(loadedDelegatorRewards)
	}
	if len(loadedWithdrawals) > 0 {
		withdrawalReg.Load(loadedWithdrawals)
	}
	if len(loadedJails) > 0 {
		jailReg.Load(loadedJails)
	}
	if len(loadedUnbondClaims) > 0 {
		unbondClaimReg.Load(loadedUnbondClaims)
	}

	bc := &Blockchain{
		blocks:                 loadedBlocks,
		accounts:               accountMgr,
		state:                  stateMgr,
		transition:             transition,
		executor:               executor,
		contractStorage:        contractStorage,
		contractRegistry:       contractRegistry,
		stateCommitment:        stateCommitment,
		mempool:                pool,
		receipts:               loadedReceipts,
		txIndex:                loadedTxIndex,
		validatorSet:           validatorSet,
		validatorReg:           registry,
		stakingReg:             stakingReg,
		delegationReg:          delegationReg,
		undelegationReg:        undelegationReg,
		slashReg:               slashReg,
		rewardReg:              rewardReg,
		delegatorRewardReg:     delegatorRewardReg,
		withdrawalReg:          withdrawalReg,
		jailReg:                jailReg,
		unbondClaimReg:         unbondClaimReg,
		activeValidators:       nil,
		weightedValidators:     nil,
		minValidatorStake:      minValidatorStake,
		blockReward:            protocolCfg.BlockReward,
		unbondingDelay:         protocolCfg.UnbondingDelay,
		validatorCommissionBps: protocolCfg.ValidatorCommissionBps,
		meta:                   loadedMeta,
		db:                     db,
		blockStore:             blockStore,
		accountStore:           accountStore,
		receiptStore:           receiptStore,
		txStore:                txStore,
		metaStore:              metaStore,
		contractCodeStore:      contractCodeStore,
		contractStorageStore:   contractStorageStore,
		stateStore:             stateStore,
		validatorStore:         validatorStore,
		stakingStore:           stakingStore,
		delegationStore:        delegationStore,
		undelegationStore:      undelegationStore,
		slashStore:             slashStore,
		rewardStore:            rewardStore,
		delegatorRewardStore:   delegatorRewardStore,
		withdrawalStore:        withdrawalStore,
		jailStore:              jailStore,
		unbondClaimStore:       unbondClaimStore,
	}

	bc.rebuildActiveValidators()

	if trieCommitment, ok := bc.stateCommitment.(*state.TrieStateCommitment); ok {
		persistedRoot, err := trieCommitment.LoadPersistedStateRoot()
		if err != nil {
			return nil, err
		}

		recomputedRoot, err := bc.stateCommitment.ComputeStateRoot(bc.accounts)
		if err != nil {
			return nil, err
		}

		latest, err := bc.LatestBlock()
		if err != nil {
			return nil, err
		}

		if persistedRoot != "" {
			latest.Header.StateRoot = persistedRoot
			if recomputedRoot != "" && persistedRoot != recomputedRoot {
				return nil, errors.New("persisted trie root mismatch after reload")
			}
		} else if recomputedRoot != "" {
			latest.Header.StateRoot = recomputedRoot
		}
	}

	if err := bc.persist(); err != nil {
		return nil, err
	}

	return bc, nil
}

func (bc *Blockchain) persist() error {
	if err := bc.blockStore.Save(bc.blocks); err != nil {
		return err
	}
	if err := bc.accountStore.Save(bc.accounts); err != nil {
		return err
	}
	if err := bc.receiptStore.Save(bc.receipts); err != nil {
		return err
	}
	if err := bc.txStore.Save(bc.txIndex); err != nil {
		return err
	}
	if err := bc.metaStore.Save(bc.meta); err != nil {
		return err
	}
	if err := bc.contractCodeStore.Save(bc.contractRegistry.AllCode()); err != nil {
		return err
	}
	if err := bc.contractStorageStore.Save(bc.contractRegistry.AllStorage()); err != nil {
		return err
	}
	if err := bc.validatorStore.Save(bc.validatorReg.Members()); err != nil {
		return err
	}
	if err := bc.stakingStore.Save(bc.stakingReg.All()); err != nil {
		return err
	}
	if err := bc.delegationStore.Save(bc.delegationReg.All()); err != nil {
		return err
	}
	if err := bc.undelegationStore.Save(bc.undelegationReg.All()); err != nil {
		return err
	}
	if err := bc.slashStore.Save(bc.slashReg.All()); err != nil {
		return err
	}
	if err := bc.rewardStore.Save(bc.rewardReg.All()); err != nil {
		return err
	}
	if err := bc.delegatorRewardStore.Save(bc.delegatorRewardReg.All()); err != nil {
		return err
	}
	if err := bc.withdrawalStore.Save(bc.withdrawalReg.All()); err != nil {
		return err
	}
	if err := bc.jailStore.Save(bc.jailReg.All()); err != nil {
		return err
	}
	if err := bc.unbondClaimStore.Save(bc.unbondClaimReg.All()); err != nil {
		return err
	}
	return nil
}

func weightUnits(totalStake uint64) int {
	switch {
	case totalStake >= 5000:
		return 5
	case totalStake >= 2000:
		return 4
	case totalStake >= 1000:
		return 3
	case totalStake >= 600:
		return 2
	default:
		return 1
	}
}

func (bc *Blockchain) rebuildActiveValidators() {
	if bc == nil || bc.validatorReg == nil || bc.stakingReg == nil {
		return
	}

	active := make([]validator.Member, 0)
	weighted := make([]validator.Member, 0)

	for _, m := range bc.validatorReg.Members() {
		if bc.jailReg != nil && bc.jailReg.IsJailed(m.Address) {
			continue
		}

		selfStake := uint64(0)
		if st, ok := bc.stakingReg.Get(m.Address); ok {
			selfStake = st.Stake
		}
		delegatedStake := uint64(0)
		if bc.delegationReg != nil {
			delegatedStake = bc.delegationReg.TotalForValidator(m.Address)
		}
		totalStake := selfStake + delegatedStake

		if totalStake >= bc.minValidatorStake {
			member := m
			member.Stake = totalStake
			active = append(active, member)

			units := weightUnits(totalStake)
			for i := 0; i < units; i++ {
				weighted = append(weighted, member)
			}
		}
	}

	bc.activeValidators = active
	bc.weightedValidators = weighted

	if len(weighted) == 0 {
		bc.meta.NextValidatorIndex = 0
		return
	}
	if bc.meta.NextValidatorIndex >= len(weighted) {
		bc.meta.NextValidatorIndex = bc.meta.NextValidatorIndex % len(weighted)
	}
}

func (bc *Blockchain) Accounts() *accounts.Manager               { return bc.accounts }
func (bc *Blockchain) State() *state.Manager                     { return bc.state }
func (bc *Blockchain) Mempool() *mempool.Pool                    { return bc.mempool }
func (bc *Blockchain) ContractRegistry() *state.ContractRegistry { return bc.contractRegistry }

func (bc *Blockchain) LatestBlock() (*coretypes.Block, error) {
	if bc == nil {
		return nil, ErrNilBlockchain
	}
	if len(bc.blocks) == 0 {
		return nil, ErrEmptyChain
	}
	return bc.blocks[len(bc.blocks)-1], nil
}

func (bc *Blockchain) Height() (pkgtypes.Height, error) {
	latest, err := bc.LatestBlock()
	if err != nil {
		return 0, err
	}
	return latest.Header.Height, nil
}

func (bc *Blockchain) Blocks() []*coretypes.Block {
	out := make([]*coretypes.Block, len(bc.blocks))
	copy(out, bc.blocks)
	return out
}

func (bc *Blockchain) RegisterAccount(address pkgtypes.Address, publicKey string) (*accounts.Account, error) {
	acc, err := bc.accounts.RegisterAccount(address, publicKey)
	if err != nil {
		return nil, err
	}
	if err := bc.persist(); err != nil {
		return nil, err
	}
	return acc, nil
}

func (bc *Blockchain) GetAccount(address pkgtypes.Address) (*accounts.Account, error) {
	acc, ok := bc.accounts.Get(address)
	if !ok {
		return nil, accounts.ErrAccountNotFound
	}
	return acc, nil
}

func (bc *Blockchain) Faucet(address pkgtypes.Address, amount pkgtypes.Amount) error {
	acc, ok := bc.accounts.Get(address)
	if !ok {
		return accounts.ErrAccountNotFound
	}
	acc.Credit(amount)
	return bc.persist()
}

func (bc *Blockchain) CreateContractAccount(address pkgtypes.Address, codeHash pkgtypes.Hash, initialBalance pkgtypes.Amount) (*accounts.Account, error) {
	if bc == nil || bc.contractRegistry == nil {
		return nil, ErrNilBlockchain
	}

	acc, err := bc.contractRegistry.CreateContractAccount(address, codeHash, initialBalance)
	if err != nil {
		return nil, err
	}

	if err := bc.persist(); err != nil {
		return nil, err
	}
	return acc, nil
}

func (bc *Blockchain) SetContractStorage(address pkgtypes.Address, key string, value string) error {
	if bc == nil || bc.contractRegistry == nil {
		return ErrNilBlockchain
	}

	if err := bc.contractRegistry.SetStorage(address, key, value); err != nil {
		return err
	}

	return bc.persist()
}

func (bc *Blockchain) GetContractStorage(address pkgtypes.Address, key string) (string, bool) {
	if bc == nil || bc.contractRegistry == nil {
		return "", false
	}

	return bc.contractRegistry.GetStorage(address, key)
}

func (bc *Blockchain) GetContractStorageRoot(address pkgtypes.Address) (pkgtypes.Hash, error) {
	if bc == nil || bc.contractRegistry == nil {
		return "", ErrNilBlockchain
	}

	return bc.contractRegistry.StorageRoot(address)
}

func (bc *Blockchain) CallContract(address pkgtypes.Address, method string, key string, value string) (state.ContractCallResult, error) {
	if bc == nil || bc.contractRegistry == nil {
		return state.ContractCallResult{}, ErrNilBlockchain
	}

	result, err := bc.contractRegistry.Call(address, method, key, value)
	if err != nil {
		return state.ContractCallResult{}, err
	}

	if result.Mutated {
		if err := bc.persist(); err != nil {
			return state.ContractCallResult{}, err
		}
	}

	return result, nil
}

func (bc *Blockchain) DeployContract(address pkgtypes.Address, code string, initialBalance pkgtypes.Amount) (*accounts.Account, pkgtypes.Hash, error) {
	if bc == nil || bc.contractRegistry == nil {
		return nil, "", ErrNilBlockchain
	}

	acc, codeHash, err := bc.contractRegistry.DeployContract(address, code, initialBalance)
	if err != nil {
		return nil, "", err
	}

	if err := bc.persist(); err != nil {
		return nil, "", err
	}

	return acc, codeHash, nil
}

func (bc *Blockchain) GetContractCode(address pkgtypes.Address) (string, bool) {
	if bc == nil || bc.contractRegistry == nil {
		return "", false
	}

	return bc.contractRegistry.GetCode(address)
}

func (bc *Blockchain) finalizeBlockRoots(blk *coretypes.Block) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	if blk == nil {
		return ErrEmptyChain
	}

	if bc.contractRegistry != nil && bc.accounts != nil {
		for _, acc := range bc.accounts.All() {
			if acc == nil {
				continue
			}

			storageRoot, err := bc.contractRegistry.StorageRoot(acc.Address)
			if err != nil {
				return err
			}

			acc.StorageRoot = storageRoot
		}
	}

	stateRoot, err := bc.stateCommitment.ComputeStateRoot(bc.accounts)
	if err != nil {
		return err
	}
	blk.Header.StateRoot = stateRoot

	txRoot, err := block.TxRootHash(blk.Transactions)
	if err != nil {
		return err
	}
	blk.Header.TxRoot = txRoot

	receiptRoot, err := block.ReceiptRootHash(blk.Receipts)
	if err != nil {
		return err
	}
	blk.Header.ReceiptRoot = receiptRoot

	hash, err := block.HeaderHash(blk.Header)
	if err != nil {
		return err
	}
	blk.Header.Hash = hash

	return nil
}

func (bc *Blockchain) finalizeAnchoredReceiptRoots(blk *coretypes.Block) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	if blk == nil {
		return ErrEmptyChain
	}

	finalReceiptRoot, err := block.ReceiptRootHash(blk.Receipts)
	if err != nil {
		return err
	}
	blk.Header.ReceiptRoot = finalReceiptRoot

	hash, err := block.HeaderHash(blk.Header)
	if err != nil {
		return err
	}
	blk.Header.Hash = hash

	return nil
}
