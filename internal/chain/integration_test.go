package chain

import (
	"crypto/ecdsa"
	"os"
	"testing"
	"time"

	block "silachain/internal/block"
	coretypes "silachain/internal/core/types"
	"silachain/internal/mempool"
	"silachain/internal/validator"
	chaincrypto "silachain/pkg/crypto"
	pkgtypes "silachain/pkg/types"
)

func mustKeyAndAddress(t *testing.T) (*ecdsa.PrivateKey, string, string) {
	t.Helper()

	priv, pub, err := chaincrypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	pubHex := chaincrypto.PublicKeyToHex(pub)
	addr := chaincrypto.PublicKeyToAddress(pub)

	return priv, pubHex, string(addr)
}

func mustNewBlockchainForTest(t *testing.T) (*Blockchain, pkgtypes.Address, *ecdsa.PrivateKey, string, pkgtypes.Address, *ecdsa.PrivateKey, string) {
	t.Helper()

	senderPriv, senderPubHex, senderAddrStr := mustKeyAndAddress(t)
	receiverPriv, receiverPubHex, receiverAddrStr := mustKeyAndAddress(t)

	validatorSet := validator.NewSet([]validator.Member{
		{
			Address:   pkgtypes.Address(senderAddrStr),
			PublicKey: senderPubHex,
			Power:     100,
			Stake:     1000,
		},
	})

	bc, err := NewBlockchain(t.TempDir(), validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain failed: %v", err)
	}

	senderAddr := pkgtypes.Address(senderAddrStr)
	receiverAddr := pkgtypes.Address(receiverAddrStr)

	if _, err := bc.RegisterAccount(senderAddr, senderPubHex); err != nil {
		t.Fatalf("RegisterAccount sender failed: %v", err)
	}
	if _, err := bc.RegisterAccount(receiverAddr, receiverPubHex); err != nil {
		t.Fatalf("RegisterAccount receiver failed: %v", err)
	}
	if err := bc.Faucet(senderAddr, 1000); err != nil {
		t.Fatalf("Faucet failed: %v", err)
	}

	return bc, senderAddr, senderPriv, senderPubHex, receiverAddr, receiverPriv, receiverPubHex
}

func mustNewBlockchainWithDelegatorValidator(t *testing.T) (*Blockchain, pkgtypes.Address, *ecdsa.PrivateKey, string, pkgtypes.Address, *ecdsa.PrivateKey, string) {
	t.Helper()

	validatorPriv, validatorPubHex, validatorAddrStr := mustKeyAndAddress(t)
	delegatorPriv, delegatorPubHex, delegatorAddrStr := mustKeyAndAddress(t)

	validatorAddr := pkgtypes.Address(validatorAddrStr)
	delegatorAddr := pkgtypes.Address(delegatorAddrStr)

	validatorSet := validator.NewSet([]validator.Member{
		{
			Address:   validatorAddr,
			PublicKey: validatorPubHex,
			Power:     100,
			Stake:     1000,
		},
	})

	bc, err := NewBlockchain(t.TempDir(), validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain failed: %v", err)
	}

	if _, err := bc.RegisterAccount(validatorAddr, validatorPubHex); err != nil {
		t.Fatalf("RegisterAccount validator failed: %v", err)
	}
	if _, err := bc.RegisterAccount(delegatorAddr, delegatorPubHex); err != nil {
		t.Fatalf("RegisterAccount delegator failed: %v", err)
	}

	if err := bc.Faucet(validatorAddr, 1000); err != nil {
		t.Fatalf("Faucet validator failed: %v", err)
	}
	if err := bc.Faucet(delegatorAddr, 1000); err != nil {
		t.Fatalf("Faucet delegator failed: %v", err)
	}

	return bc, validatorAddr, validatorPriv, validatorPubHex, delegatorAddr, delegatorPriv, delegatorPubHex
}

func mustSignedTx(
	t *testing.T,
	from pkgtypes.Address,
	to pkgtypes.Address,
	nonce pkgtypes.Nonce,
	value pkgtypes.Amount,
	fee pkgtypes.Amount,
	chainID pkgtypes.ChainID,
	publicKey string,
	priv *ecdsa.PrivateKey,
) *coretypes.Transaction {
	t.Helper()

	txn := &coretypes.Transaction{
		From:      from,
		To:        to,
		Value:     value,
		Fee:       fee,
		Nonce:     nonce,
		ChainID:   chainID,
		Timestamp: pkgtypes.Timestamp(time.Now().Unix()),
		PublicKey: publicKey,
	}

	if err := coretypes.SignTransaction(txn, priv); err != nil {
		t.Fatalf("SignTransaction failed: %v", err)
	}

	return txn
}

func TestSubmitTransaction_Success(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	txn := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	if err := bc.SubmitTransaction(txn); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}

	if got := bc.Mempool().Count(); got != 1 {
		t.Fatalf("expected mempool count 1, got %d", got)
	}
}

func TestSubmitTransaction_DuplicateSenderNonce(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 5, 1001, senderPubHex, senderPriv)
	tx2 := mustSignedTx(t, senderAddr, receiverAddr, 0, 11, 1, 1001, senderPubHex, senderPriv)

	if err := bc.SubmitTransaction(tx1); err != nil {
		t.Fatalf("first SubmitTransaction failed: %v", err)
	}

	err := bc.SubmitTransaction(tx2)
	if err == nil {
		t.Fatalf("expected duplicate sender nonce error, got nil")
	}
	if err.Error() != "duplicate sender nonce" {
		t.Fatalf("expected duplicate sender nonce, got %v", err)
	}

	if got := bc.Mempool().Count(); got != 1 {
		t.Fatalf("expected mempool count 1, got %d", got)
	}
}

func TestSubmitTransaction_InvalidNonce(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 5, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(tx1); err != nil {
		t.Fatalf("SubmitTransaction tx1 failed: %v", err)
	}
	if _, err := bc.MinePending(senderAddr); err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}

	staleTx := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	staleTx.Timestamp = pkgtypes.Timestamp(time.Now().Unix() + 1)
	staleTx.Signature = ""
	staleTx.Hash = ""

	if err := coretypes.SignTransaction(staleTx, senderPriv); err != nil {
		t.Fatalf("SignTransaction staleTx failed: %v", err)
	}

	err := bc.SubmitTransaction(staleTx)
	if err != coretypes.ErrInvalidNonce {
		t.Fatalf("expected invalid nonce error, got %v", err)
	}
}

func TestMinePending_ClearsMempoolAndAdvancesHeight(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	beforeHeight, err := bc.Height()
	if err != nil {
		t.Fatalf("Height before failed: %v", err)
	}

	txn := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	if err := bc.SubmitTransaction(txn); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}

	mined, err := bc.MinePending(senderAddr)
	if err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}
	if mined == nil {
		t.Fatalf("expected mined block, got nil")
	}

	afterHeight, err := bc.Height()
	if err != nil {
		t.Fatalf("Height after failed: %v", err)
	}

	if afterHeight != beforeHeight+1 {
		t.Fatalf("expected height %d, got %d", beforeHeight+1, afterHeight)
	}

	if got := bc.Mempool().Count(); got != 0 {
		t.Fatalf("expected mempool count 0, got %d", got)
	}

	senderAcc, err := bc.GetAccount(senderAddr)
	if err != nil {
		t.Fatalf("GetAccount sender failed: %v", err)
	}
	if senderAcc.Nonce != 1 {
		t.Fatalf("expected sender nonce 1, got %d", senderAcc.Nonce)
	}

	receiverAcc, err := bc.GetAccount(receiverAddr)
	if err != nil {
		t.Fatalf("GetAccount receiver failed: %v", err)
	}
	if receiverAcc.Balance != 10 {
		t.Fatalf("expected receiver balance 10, got %d", receiverAcc.Balance)
	}
}

func TestAddDelegation_Success(t *testing.T) {
	bc, validatorAddr, _, _, delegatorAddr, _, _ := mustNewBlockchainWithDelegatorValidator(t)

	if err := bc.AddDelegation(delegatorAddr, validatorAddr, 20); err != nil {
		t.Fatalf("AddDelegation failed: %v", err)
	}

	ds := bc.Delegations()
	if len(ds) != 1 {
		t.Fatalf("expected 1 delegation, got %d", len(ds))
	}
	if ds[0].Delegator != delegatorAddr || ds[0].Validator != validatorAddr || ds[0].Amount != 20 {
		t.Fatalf("unexpected delegation: %+v", ds[0])
	}
}

func TestUndelegate_CreatesPendingUnbond(t *testing.T) {
	bc, validatorAddr, _, _, delegatorAddr, _, _ := mustNewBlockchainWithDelegatorValidator(t)

	if err := bc.AddDelegation(delegatorAddr, validatorAddr, 20); err != nil {
		t.Fatalf("AddDelegation failed: %v", err)
	}

	if err := bc.Undelegate(delegatorAddr, validatorAddr, 5, "test undelegate"); err != nil {
		t.Fatalf("Undelegate failed: %v", err)
	}

	ds := bc.Delegations()
	if len(ds) != 1 || ds[0].Amount != 15 {
		t.Fatalf("expected remaining delegation 15, got %+v", ds)
	}

	us := bc.Undelegations()
	if len(us) != 1 {
		t.Fatalf("expected 1 undelegation, got %d", len(us))
	}
	if us[0].Amount != 5 {
		t.Fatalf("expected undelegation amount 5, got %d", us[0].Amount)
	}

	if got := bc.PendingUnbond(delegatorAddr); got != 0 {
		t.Fatalf("expected pending unbond 0 before unlock, got %d", got)
	}
}

func TestClaimUnbond_AfterDelay(t *testing.T) {
	bc, validatorAddr, _, _, delegatorAddr, _, _ := mustNewBlockchainWithDelegatorValidator(t)

	if err := bc.AddDelegation(delegatorAddr, validatorAddr, 20); err != nil {
		t.Fatalf("AddDelegation failed: %v", err)
	}

	if err := bc.Undelegate(delegatorAddr, validatorAddr, 5, "test undelegate"); err != nil {
		t.Fatalf("Undelegate failed: %v", err)
	}

	u := bc.Undelegations()
	if len(u) != 1 {
		t.Fatalf("expected 1 undelegation, got %d", len(u))
	}

	currentHeight, err := bc.Height()
	if err != nil {
		t.Fatalf("Height failed: %v", err)
	}

	bc.blocks[len(bc.blocks)-1].Header.Height = pkgtypes.Height(u[0].UnlockHeight)
	defer func() {
		bc.blocks[len(bc.blocks)-1].Header.Height = currentHeight
	}()

	if got := bc.PendingUnbond(delegatorAddr); got != 5 {
		t.Fatalf("expected pending unbond 5 after unlock, got %d", got)
	}

	beforeAcc, err := bc.GetAccount(delegatorAddr)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	beforeBal := beforeAcc.Balance

	claimed, err := bc.ClaimUnbond(delegatorAddr)
	if err != nil {
		t.Fatalf("ClaimUnbond failed: %v", err)
	}
	if claimed != 5 {
		t.Fatalf("expected claimed 5, got %d", claimed)
	}

	afterAcc, err := bc.GetAccount(delegatorAddr)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if afterAcc.Balance != beforeBal+5 {
		t.Fatalf("expected balance %d, got %d", beforeBal+5, afterAcc.Balance)
	}
}

func TestMinePending_CreatesValidatorAndDelegatorRewards(t *testing.T) {
	bc, validatorAddr, validatorPriv, validatorPubHex, delegatorAddr, _, _ := mustNewBlockchainWithDelegatorValidator(t)

	if err := bc.AddDelegation(delegatorAddr, validatorAddr, 20); err != nil {
		t.Fatalf("AddDelegation failed: %v", err)
	}

	txn := mustSignedTx(t, validatorAddr, delegatorAddr, 0, 10, 1, 1001, validatorPubHex, validatorPriv)
	if err := bc.SubmitTransaction(txn); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}

	if _, err := bc.MinePending(validatorAddr); err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}

	rewards := bc.Rewards()
	if len(rewards) == 0 {
		t.Fatalf("expected validator rewards")
	}

	delegatorRewards := bc.DelegatorRewards()
	if len(delegatorRewards) == 0 {
		t.Fatalf("expected delegator rewards")
	}

	if rewards[len(rewards)-1].Validator != validatorAddr {
		t.Fatalf("unexpected reward validator: %+v", rewards[len(rewards)-1])
	}
	if delegatorRewards[len(delegatorRewards)-1].Delegator != delegatorAddr {
		t.Fatalf("unexpected delegator reward: %+v", delegatorRewards[len(delegatorRewards)-1])
	}
}

func TestCommission_SplitApplied(t *testing.T) {
	bc, validatorAddr, validatorPriv, validatorPubHex, delegatorAddr, _, _ := mustNewBlockchainWithDelegatorValidator(t)

	if err := bc.AddDelegation(delegatorAddr, validatorAddr, 20); err != nil {
		t.Fatalf("AddDelegation failed: %v", err)
	}

	txn := mustSignedTx(t, validatorAddr, delegatorAddr, 0, 10, 1, 1001, validatorPubHex, validatorPriv)
	if err := bc.SubmitTransaction(txn); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}

	if _, err := bc.MinePending(validatorAddr); err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}

	rewards := bc.Rewards()
	delegatorRewards := bc.DelegatorRewards()

	if len(rewards) == 0 || len(delegatorRewards) == 0 {
		t.Fatalf("expected both validator and delegator rewards")
	}

	validatorReward := rewards[len(rewards)-1].Amount
	delegatorReward := delegatorRewards[len(delegatorRewards)-1].Amount

	if validatorReward != 1 {
		t.Fatalf("expected validator commission reward 1, got %d", validatorReward)
	}
	if delegatorReward != 9 {
		t.Fatalf("expected delegator reward 9, got %d", delegatorReward)
	}
}

func TestAddSlash_ReducesDelegations(t *testing.T) {
	bc, validatorAddr, _, _, delegatorAddr, _, _ := mustNewBlockchainWithDelegatorValidator(t)

	if err := bc.SetStake(validatorAddr, 100); err != nil {
		t.Fatalf("SetStake failed: %v", err)
	}
	if err := bc.AddDelegation(delegatorAddr, validatorAddr, 20); err != nil {
		t.Fatalf("AddDelegation failed: %v", err)
	}

	if err := bc.AddSlash(validatorAddr, 50, "test slash"); err != nil {
		t.Fatalf("AddSlash failed: %v", err)
	}

	ds := bc.Delegations()
	if len(ds) != 1 {
		t.Fatalf("expected 1 delegation after slash, got %d", len(ds))
	}

	if ds[0].Amount != 10 {
		t.Fatalf("expected delegated amount reduced to 10, got %d", ds[0].Amount)
	}
}

func TestJailAndUnjail_AffectActiveValidators(t *testing.T) {
	bc, validatorAddr, _, _, _, _, _ := mustNewBlockchainWithDelegatorValidator(t)

	activeBefore := bc.ActiveValidators()
	if len(activeBefore) == 0 {
		t.Fatalf("expected active validators before jail")
	}

	if err := bc.JailValidator(validatorAddr, "test jail"); err != nil {
		t.Fatalf("JailValidator failed: %v", err)
	}

	activeAfterJail := bc.ActiveValidators()
	if len(activeAfterJail) != 0 {
		t.Fatalf("expected 0 active validators after jail, got %d", len(activeAfterJail))
	}

	if err := bc.UnjailValidator(validatorAddr, "test unjail"); err != nil {
		t.Fatalf("UnjailValidator failed: %v", err)
	}

	activeAfterUnjail := bc.ActiveValidators()
	if len(activeAfterUnjail) == 0 {
		t.Fatalf("expected active validators after unjail")
	}
}

func mustNewBlockchainForTestAtDir(t *testing.T, dataDir string) (*Blockchain, pkgtypes.Address, *ecdsa.PrivateKey, string, pkgtypes.Address, *ecdsa.PrivateKey, string, *validator.Set) {
	t.Helper()

	senderPriv, senderPubHex, senderAddrStr := mustKeyAndAddress(t)
	receiverPriv, receiverPubHex, receiverAddrStr := mustKeyAndAddress(t)

	validatorSet := validator.NewSet([]validator.Member{
		{
			Address:   pkgtypes.Address(senderAddrStr),
			PublicKey: senderPubHex,
			Power:     100,
			Stake:     1000,
		},
	})

	bc, err := NewBlockchain(dataDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain failed: %v", err)
	}

	senderAddr := pkgtypes.Address(senderAddrStr)
	receiverAddr := pkgtypes.Address(receiverAddrStr)

	if _, err := bc.RegisterAccount(senderAddr, senderPubHex); err != nil {
		t.Fatalf("RegisterAccount sender failed: %v", err)
	}
	if _, err := bc.RegisterAccount(receiverAddr, receiverPubHex); err != nil {
		t.Fatalf("RegisterAccount receiver failed: %v", err)
	}
	if err := bc.Faucet(senderAddr, 1000); err != nil {
		t.Fatalf("Faucet failed: %v", err)
	}

	return bc, senderAddr, senderPriv, senderPubHex, receiverAddr, receiverPriv, receiverPubHex, validatorSet
}

func mustNewBlockchainWithDelegatorValidatorAtDir(t *testing.T, dataDir string) (*Blockchain, pkgtypes.Address, *ecdsa.PrivateKey, string, pkgtypes.Address, *ecdsa.PrivateKey, string, *validator.Set) {
	t.Helper()

	validatorPriv, validatorPubHex, validatorAddrStr := mustKeyAndAddress(t)
	delegatorPriv, delegatorPubHex, delegatorAddrStr := mustKeyAndAddress(t)

	validatorAddr := pkgtypes.Address(validatorAddrStr)
	delegatorAddr := pkgtypes.Address(delegatorAddrStr)

	validatorSet := validator.NewSet([]validator.Member{
		{
			Address:   validatorAddr,
			PublicKey: validatorPubHex,
			Power:     100,
			Stake:     1000,
		},
	})

	bc, err := NewBlockchain(dataDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain failed: %v", err)
	}

	if _, err := bc.RegisterAccount(validatorAddr, validatorPubHex); err != nil {
		t.Fatalf("RegisterAccount validator failed: %v", err)
	}
	if _, err := bc.RegisterAccount(delegatorAddr, delegatorPubHex); err != nil {
		t.Fatalf("RegisterAccount delegator failed: %v", err)
	}

	if err := bc.Faucet(validatorAddr, 1000); err != nil {
		t.Fatalf("Faucet validator failed: %v", err)
	}
	if err := bc.Faucet(delegatorAddr, 1000); err != nil {
		t.Fatalf("Faucet delegator failed: %v", err)
	}

	return bc, validatorAddr, validatorPriv, validatorPubHex, delegatorAddr, delegatorPriv, delegatorPubHex, validatorSet
}

func TestReloadBlockchain_PreservesHeightTxIndexAndReceipts(t *testing.T) {
	dataDir := t.TempDir()

	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _, validatorSet := mustNewBlockchainForTestAtDir(t, dataDir)

	txn := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	if err := bc.SubmitTransaction(txn); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}
	mined, err := bc.MinePending(senderAddr)
	if err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}

	beforeHeight, err := bc.Height()
	if err != nil {
		t.Fatalf("Height before reload failed: %v", err)
	}

	reloaded, err := NewBlockchain(dataDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain reload failed: %v", err)
	}

	afterHeight, err := reloaded.Height()
	if err != nil {
		t.Fatalf("Height after reload failed: %v", err)
	}

	if afterHeight != beforeHeight {
		t.Fatalf("expected height %d after reload, got %d", beforeHeight, afterHeight)
	}

	gotTx, gotHeight, ok := reloaded.GetTransactionByHash(string(txn.Hash))
	if !ok || gotTx == nil {
		t.Fatalf("expected tx index after reload")
	}
	if gotHeight != mined.Header.Height {
		t.Fatalf("expected tx height %d, got %d", mined.Header.Height, gotHeight)
	}

	receipt, ok := reloaded.GetReceiptByHash(string(txn.Hash))
	if !ok || receipt == nil {
		t.Fatalf("expected receipt after reload")
	}
	if receipt.BlockHeight != mined.Header.Height {
		t.Fatalf("expected receipt block height %d, got %d", mined.Header.Height, receipt.BlockHeight)
	}
}

func TestReloadBlockchain_PreservesAccountStateAfterMining(t *testing.T) {
	dataDir := t.TempDir()

	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _, validatorSet := mustNewBlockchainForTestAtDir(t, dataDir)

	txn := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	if err := bc.SubmitTransaction(txn); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}
	if _, err := bc.MinePending(senderAddr); err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}

	reloaded, err := NewBlockchain(dataDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain reload failed: %v", err)
	}

	senderAcc, err := reloaded.GetAccount(senderAddr)
	if err != nil {
		t.Fatalf("GetAccount sender failed: %v", err)
	}
	if senderAcc.Nonce != 1 {
		t.Fatalf("expected sender nonce 1 after reload, got %d", senderAcc.Nonce)
	}

	receiverAcc, err := reloaded.GetAccount(receiverAddr)
	if err != nil {
		t.Fatalf("GetAccount receiver failed: %v", err)
	}
	if receiverAcc.Balance != 10 {
		t.Fatalf("expected receiver balance 10 after reload, got %d", receiverAcc.Balance)
	}
}

func TestReloadBlockchain_PreservesDelegationUndelegationAndJail(t *testing.T) {
	dataDir := t.TempDir()

	bc, validatorAddr, _, _, delegatorAddr, _, _, validatorSet := mustNewBlockchainWithDelegatorValidatorAtDir(t, dataDir)

	if err := bc.AddDelegation(delegatorAddr, validatorAddr, 20); err != nil {
		t.Fatalf("AddDelegation failed: %v", err)
	}
	if err := bc.Undelegate(delegatorAddr, validatorAddr, 5, "test undelegate"); err != nil {
		t.Fatalf("Undelegate failed: %v", err)
	}
	if err := bc.JailValidator(validatorAddr, "test jail"); err != nil {
		t.Fatalf("JailValidator failed: %v", err)
	}

	reloaded, err := NewBlockchain(dataDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain reload failed: %v", err)
	}

	ds := reloaded.Delegations()
	if len(ds) != 1 || ds[0].Amount != 15 {
		t.Fatalf("expected delegation amount 15 after reload, got %+v", ds)
	}

	us := reloaded.Undelegations()
	if len(us) != 1 || us[0].Amount != 5 {
		t.Fatalf("expected undelegation amount 5 after reload, got %+v", us)
	}

	jails := reloaded.Jails()
	if len(jails) != 1 {
		t.Fatalf("expected 1 jail record after reload, got %d", len(jails))
	}
	if !jails[0].Jailed {
		t.Fatalf("expected validator to remain jailed after reload")
	}

	active := reloaded.ActiveValidators()
	if len(active) != 0 {
		t.Fatalf("expected 0 active validators after reload while jailed, got %d", len(active))
	}
}

func TestReloadBlockchain_PreservesRewardsAndDelegatorRewards(t *testing.T) {
	dataDir := t.TempDir()

	bc, validatorAddr, validatorPriv, validatorPubHex, delegatorAddr, _, _, validatorSet := mustNewBlockchainWithDelegatorValidatorAtDir(t, dataDir)

	if err := bc.AddDelegation(delegatorAddr, validatorAddr, 20); err != nil {
		t.Fatalf("AddDelegation failed: %v", err)
	}

	txn := mustSignedTx(t, validatorAddr, delegatorAddr, 0, 10, 1, 1001, validatorPubHex, validatorPriv)
	if err := bc.SubmitTransaction(txn); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}
	if _, err := bc.MinePending(validatorAddr); err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}

	reloaded, err := NewBlockchain(dataDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain reload failed: %v", err)
	}

	rewards := reloaded.Rewards()
	if len(rewards) == 0 {
		t.Fatalf("expected validator rewards after reload")
	}

	delegatorRewards := reloaded.DelegatorRewards()
	if len(delegatorRewards) == 0 {
		t.Fatalf("expected delegator rewards after reload")
	}

	if rewards[len(rewards)-1].Amount != 1 {
		t.Fatalf("expected validator reward 1 after reload, got %d", rewards[len(rewards)-1].Amount)
	}
	if delegatorRewards[len(delegatorRewards)-1].Amount != 9 {
		t.Fatalf("expected delegator reward 9 after reload, got %d", delegatorRewards[len(delegatorRewards)-1].Amount)
	}
}

func TestAddBlock_RejectsInvalidHeight(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	txn := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	latest, err := bc.LatestBlock()
	if err != nil {
		t.Fatalf("LatestBlock failed: %v", err)
	}

	b, err := block.NewBlock(
		latest.Header.Height+2,
		latest.Header.Hash,
		latest.Header.StateRoot,
		"",
		"",
		senderAddr,
		0,
		0,
		[]coretypes.Transaction{*txn},
		nil,
	)
	if err != nil {
		t.Fatalf("NewBlock failed: %v", err)
	}

	err = bc.AddBlock(b)
	if err == nil {
		t.Fatalf("expected invalid height error, got nil")
	}
	if err != ErrInvalidHeight {
		t.Fatalf("expected ErrInvalidHeight, got %v", err)
	}
}

func TestAddBlock_RejectsInvalidParentHash(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	txn := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	latest, err := bc.LatestBlock()
	if err != nil {
		t.Fatalf("LatestBlock failed: %v", err)
	}

	b, err := block.NewBlock(
		latest.Header.Height+1,
		pkgtypes.Hash("bad_parent_hash"),
		latest.Header.StateRoot,
		"",
		"",
		senderAddr,
		0,
		0,
		[]coretypes.Transaction{*txn},
		nil,
	)
	if err != nil {
		t.Fatalf("NewBlock failed: %v", err)
	}

	err = bc.AddBlock(b)
	if err == nil {
		t.Fatalf("expected invalid parent error, got nil")
	}
	if err != ErrInvalidParent {
		t.Fatalf("expected ErrInvalidParent, got %v", err)
	}
}

func TestValidateLoadedBlocks_DetectsCorruptHeightSequence(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 5, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(tx1); err != nil {
		t.Fatalf("SubmitTransaction tx1 failed: %v", err)
	}
	b1, err := bc.MinePending(senderAddr)
	if err != nil {
		t.Fatalf("MinePending b1 failed: %v", err)
	}

	tx2 := mustSignedTx(t, senderAddr, receiverAddr, 1, 10, 1, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(tx2); err != nil {
		t.Fatalf("SubmitTransaction tx2 failed: %v", err)
	}
	b2, err := bc.MinePending(senderAddr)
	if err != nil {
		t.Fatalf("MinePending b2 failed: %v", err)
	}

	bad := *b2
	bad.Header.Height = b1.Header.Height + 2
	badHash, err := block.HeaderHash(bad.Header)
	if err != nil {
		t.Fatalf("HeaderHash failed: %v", err)
	}
	bad.Header.Hash = badHash

	err = validateLoadedBlocks([]*coretypes.Block{bc.blocks[0], b1, &bad})
	if err == nil {
		t.Fatalf("expected corrupt chain data error, got nil")
	}
	if err != ErrCorruptChainData {
		t.Fatalf("expected ErrCorruptChainData, got %v", err)
	}
}

func TestValidateLoadedBlocks_DetectsCorruptParentSequence(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 5, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(tx1); err != nil {
		t.Fatalf("SubmitTransaction tx1 failed: %v", err)
	}
	b1, err := bc.MinePending(senderAddr)
	if err != nil {
		t.Fatalf("MinePending b1 failed: %v", err)
	}

	tx2 := mustSignedTx(t, senderAddr, receiverAddr, 1, 10, 1, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(tx2); err != nil {
		t.Fatalf("SubmitTransaction tx2 failed: %v", err)
	}
	b2, err := bc.MinePending(senderAddr)
	if err != nil {
		t.Fatalf("MinePending b2 failed: %v", err)
	}

	bad := *b2
	bad.Header.ParentHash = pkgtypes.Hash("bad_parent_hash")
	badHash, err := block.HeaderHash(bad.Header)
	if err != nil {
		t.Fatalf("HeaderHash failed: %v", err)
	}
	bad.Header.Hash = badHash

	err = validateLoadedBlocks([]*coretypes.Block{bc.blocks[0], b1, &bad})
	if err == nil {
		t.Fatalf("expected corrupt chain data error, got nil")
	}
	if err != ErrCorruptChainData {
		t.Fatalf("expected ErrCorruptChainData, got %v", err)
	}
}

func TestForkDetection_SameHeightDifferentHash(t *testing.T) {
	bcA, senderAddrA, senderPrivA, senderPubHexA, receiverAddrA, _, _ := mustNewBlockchainForTest(t)
	bcB, senderAddrB, senderPrivB, senderPubHexB, receiverAddrB, _, _ := mustNewBlockchainForTest(t)

	txA := mustSignedTx(t, senderAddrA, receiverAddrA, 0, 10, 1, 1001, senderPubHexA, senderPrivA)
	if err := bcA.SubmitTransaction(txA); err != nil {
		t.Fatalf("SubmitTransaction A failed: %v", err)
	}
	if _, err := bcA.MinePending(senderAddrA); err != nil {
		t.Fatalf("MinePending A failed: %v", err)
	}

	txB := mustSignedTx(t, senderAddrB, receiverAddrB, 0, 20, 1, 1001, senderPubHexB, senderPrivB)
	if err := bcB.SubmitTransaction(txB); err != nil {
		t.Fatalf("SubmitTransaction B failed: %v", err)
	}
	if _, err := bcB.MinePending(senderAddrB); err != nil {
		t.Fatalf("MinePending B failed: %v", err)
	}

	heightA, err := bcA.Height()
	if err != nil {
		t.Fatalf("Height A failed: %v", err)
	}
	heightB, err := bcB.Height()
	if err != nil {
		t.Fatalf("Height B failed: %v", err)
	}

	if heightA != heightB {
		t.Fatalf("expected same height, got A=%d B=%d", heightA, heightB)
	}

	latestA, err := bcA.LatestBlock()
	if err != nil {
		t.Fatalf("LatestBlock A failed: %v", err)
	}
	latestB, err := bcB.LatestBlock()
	if err != nil {
		t.Fatalf("LatestBlock B failed: %v", err)
	}

	if latestA.Header.Hash == latestB.Header.Hash {
		t.Fatalf("expected different hashes for fork detection, got same hash %s", latestA.Header.Hash)
	}
}

func TestReorgGuard_BlocksDivergentNextBlock(t *testing.T) {
	bcLocal, senderAddrLocal, senderPrivLocal, senderPubHexLocal, receiverAddrLocal, _, _ := mustNewBlockchainForTest(t)
	bcPeer, senderAddrPeer, senderPrivPeer, senderPubHexPeer, receiverAddrPeer, _, _ := mustNewBlockchainForTest(t)

	txLocal := mustSignedTx(t, senderAddrLocal, receiverAddrLocal, 0, 10, 1, 1001, senderPubHexLocal, senderPrivLocal)
	if err := bcLocal.SubmitTransaction(txLocal); err != nil {
		t.Fatalf("SubmitTransaction local failed: %v", err)
	}
	localBlock, err := bcLocal.MinePending(senderAddrLocal)
	if err != nil {
		t.Fatalf("MinePending local failed: %v", err)
	}

	txPeer := mustSignedTx(t, senderAddrPeer, receiverAddrPeer, 0, 20, 1, 1001, senderPubHexPeer, senderPrivPeer)
	if err := bcPeer.SubmitTransaction(txPeer); err != nil {
		t.Fatalf("SubmitTransaction peer failed: %v", err)
	}
	peerBlock, err := bcPeer.MinePending(senderAddrPeer)
	if err != nil {
		t.Fatalf("MinePending peer failed: %v", err)
	}

	genesisLocal, ok := bcLocal.GetBlockByHeight(0)
	if !ok {
		t.Fatalf("missing local genesis")
	}
	genesisPeer, ok := bcPeer.GetBlockByHeight(0)
	if !ok {
		t.Fatalf("missing peer genesis")
	}

	if genesisLocal.Header.Hash != genesisPeer.Header.Hash {
		t.Fatalf("expected same genesis hash")
	}
	if localBlock.Header.Height != peerBlock.Header.Height {
		t.Fatalf("expected same competing height")
	}
	if localBlock.Header.Hash == peerBlock.Header.Hash {
		t.Fatalf("expected divergent competing blocks")
	}
	if peerBlock.Header.ParentHash != genesisLocal.Header.Hash {
		t.Fatalf("expected peer block parent to equal local tip before divergence")
	}

	err = bcLocal.AddBlock(peerBlock)
	if err == nil {
		t.Fatalf("expected add block to fail on divergent competing block")
	}
	if err != ErrInvalidParent && err != ErrInvalidHeight && err != ErrInvalidProposerTurn {
		t.Fatalf("expected ErrInvalidParent or ErrInvalidHeight or ErrInvalidProposerTurn, got %v", err)
	}
}

func TestSubmitTransaction_RejectsTooOldTimestamp(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	oldTs := time.Now().Add(-48 * time.Hour).Unix()
	transaction := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	transaction.Timestamp = pkgtypes.Timestamp(oldTs)
	transaction.Signature = ""
	transaction.Hash = ""
	if err := coretypes.SignTransaction(transaction, senderPriv); err != nil {
		t.Fatalf("SignTransaction failed: %v", err)
	}

	err := bc.SubmitTransaction(transaction)
	if err != coretypes.ErrInvalidTimestamp {
		t.Fatalf("expected ErrInvalidTimestamp, got %v", err)
	}
}

func TestSubmitTransaction_RejectsTooFutureTimestamp(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	futureTs := time.Now().Add(2 * time.Hour).Unix()
	transaction := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	transaction.Timestamp = pkgtypes.Timestamp(futureTs)
	transaction.Signature = ""
	transaction.Hash = ""
	if err := coretypes.SignTransaction(transaction, senderPriv); err != nil {
		t.Fatalf("SignTransaction failed: %v", err)
	}

	err := bc.SubmitTransaction(transaction)
	if err != coretypes.ErrInvalidTimestamp {
		t.Fatalf("expected ErrInvalidTimestamp, got %v", err)
	}
}

func TestMinePending_AllowsConsensusProvidedProposer(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	transaction := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(transaction); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}

	var wrong pkgtypes.Address
	for wrong == "" || wrong == senderAddr {
		_, _, addrStr := mustKeyAndAddress(t)
		wrong = pkgtypes.Address(addrStr)
	}

	if _, err := bc.MinePending(wrong); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestSubmitTransaction_RejectsTooFarFutureNonce(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	transaction := mustSignedTx(t, senderAddr, receiverAddr, 20, 10, 1, 1001, senderPubHex, senderPriv)

	err := bc.SubmitTransaction(transaction)
	if err == nil {
		t.Fatalf("expected nonce-too-far error, got nil")
	}
	if err.Error() != "nonce too far in future" {
		t.Fatalf("expected nonce too far in future, got %v", err)
	}
}

func TestSubmitTransaction_RejectsSenderQuotaExceeded(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	bc.Mempool().SetLimits(1024, 2, 8)

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 5, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(tx1); err != nil {
		t.Fatalf("SubmitTransaction tx1 failed: %v", err)
	}

	tx2 := mustSignedTx(t, senderAddr, receiverAddr, 1, 10, 1, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(tx2); err != nil {
		t.Fatalf("SubmitTransaction tx2 failed: %v", err)
	}

	tx3 := mustSignedTx(t, senderAddr, receiverAddr, 2, 10, 1, 1001, senderPubHex, senderPriv)
	err := bc.SubmitTransaction(tx3)
	if err == nil {
		t.Fatalf("expected sender quota error, got nil")
	}
	if err.Error() != "sender mempool quota exceeded" {
		t.Fatalf("expected sender quota error, got %v", err)
	}
}

func TestSubmitTransaction_RejectsPoolFull(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, receiverPriv, receiverPubHex := mustNewBlockchainForTest(t)

	thirdPriv, thirdPubHex, thirdAddrStr := mustKeyAndAddress(t)
	thirdAddr := pkgtypes.Address(thirdAddrStr)

	if _, err := bc.RegisterAccount(thirdAddr, thirdPubHex); err != nil {
		t.Fatalf("RegisterAccount third failed: %v", err)
	}
	if err := bc.Faucet(thirdAddr, 1000); err != nil {
		t.Fatalf("Faucet third failed: %v", err)
	}

	bc.Mempool().SetLimits(2, 16, 8)

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 5, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(tx1); err != nil {
		t.Fatalf("SubmitTransaction tx1 failed: %v", err)
	}

	tx2 := mustSignedTx(t, receiverAddr, senderAddr, 0, 10, 6, 1001, receiverPubHex, receiverPriv)
	if err := bc.Faucet(receiverAddr, 1000); err != nil {
		t.Fatalf("Faucet receiver failed: %v", err)
	}
	if err := bc.SubmitTransaction(tx2); err != nil {
		t.Fatalf("SubmitTransaction tx2 failed: %v", err)
	}

	tx3 := mustSignedTx(t, thirdAddr, senderAddr, 0, 10, 1, 1001, thirdPubHex, thirdPriv)
	err := bc.SubmitTransaction(tx3)
	if err == nil {
		t.Fatalf("expected pool full error, got nil")
	}
	if err.Error() != "mempool is full" {
		t.Fatalf("expected pool full error, got %v", err)
	}
}

func TestOrderedPending_PrioritizesHigherFeeAcrossSenders(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, receiverPriv, receiverPubHex := mustNewBlockchainForTest(t)

	secondPriv, secondPubHex, secondAddrStr := mustKeyAndAddress(t)
	secondAddr := pkgtypes.Address(secondAddrStr)

	if _, err := bc.RegisterAccount(secondAddr, secondPubHex); err != nil {
		t.Fatalf("RegisterAccount second failed: %v", err)
	}
	if err := bc.Faucet(secondAddr, 1000); err != nil {
		t.Fatalf("Faucet second failed: %v", err)
	}
	if err := bc.Faucet(receiverAddr, 1000); err != nil {
		t.Fatalf("Faucet receiver failed: %v", err)
	}

	lowFee := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	highFee := mustSignedTx(t, secondAddr, senderAddr, 0, 10, 9, 1001, secondPubHex, secondPriv)
	midFee := mustSignedTx(t, receiverAddr, secondAddr, 0, 10, 5, 1001, receiverPubHex, receiverPriv)

	if err := bc.SubmitTransaction(lowFee); err != nil {
		t.Fatalf("SubmitTransaction lowFee failed: %v", err)
	}
	if err := bc.SubmitTransaction(highFee); err != nil {
		t.Fatalf("SubmitTransaction highFee failed: %v", err)
	}
	if err := bc.SubmitTransaction(midFee); err != nil {
		t.Fatalf("SubmitTransaction midFee failed: %v", err)
	}

	ordered := bc.Mempool().Pending()
	if len(ordered) != 3 {
		t.Fatalf("expected 3 pending txs, got %d", len(ordered))
	}

	sorted := mempool.OrderedPending(bc.Mempool())
	if len(sorted) != 3 {
		t.Fatalf("expected 3 ordered txs, got %d", len(sorted))
	}

	if sorted[0].Hash != highFee.Hash {
		t.Fatalf("expected highest fee tx first, got %s", sorted[0].Hash)
	}
	if sorted[1].Hash != midFee.Hash {
		t.Fatalf("expected middle fee tx second, got %s", sorted[1].Hash)
	}
	if sorted[2].Hash != lowFee.Hash {
		t.Fatalf("expected lowest fee tx last, got %s", sorted[2].Hash)
	}
}

func TestOrderedPending_PreservesSenderNonceOrder(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 5, 1001, senderPubHex, senderPriv)
	tx2 := mustSignedTx(t, senderAddr, receiverAddr, 1, 10, 9, 1001, senderPubHex, senderPriv)

	if err := bc.SubmitTransaction(tx1); err != nil {
		t.Fatalf("SubmitTransaction tx1 failed: %v", err)
	}
	if err := bc.SubmitTransaction(tx2); err != nil {
		t.Fatalf("SubmitTransaction tx2 failed: %v", err)
	}

	sorted := mempool.OrderedPending(bc.Mempool())
	if len(sorted) != 2 {
		t.Fatalf("expected 2 ordered txs, got %d", len(sorted))
	}

	if sorted[0].Nonce != 0 || sorted[1].Nonce != 1 {
		t.Fatalf("expected sender nonce order 0 then 1, got %d then %d", sorted[0].Nonce, sorted[1].Nonce)
	}
}

func TestSubmitTransaction_EvictsLowerFeeTailWhenPoolFull(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, receiverPriv, receiverPubHex := mustNewBlockchainForTest(t)

	thirdPriv, thirdPubHex, thirdAddrStr := mustKeyAndAddress(t)
	thirdAddr := pkgtypes.Address(thirdAddrStr)

	if _, err := bc.RegisterAccount(thirdAddr, thirdPubHex); err != nil {
		t.Fatalf("RegisterAccount third failed: %v", err)
	}
	if err := bc.Faucet(thirdAddr, 1000); err != nil {
		t.Fatalf("Faucet third failed: %v", err)
	}
	if err := bc.Faucet(receiverAddr, 1000); err != nil {
		t.Fatalf("Faucet receiver failed: %v", err)
	}

	bc.Mempool().SetLimits(2, 16, 8)

	lowFee := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	midFee := mustSignedTx(t, receiverAddr, senderAddr, 0, 10, 3, 1001, receiverPubHex, receiverPriv)

	if err := bc.SubmitTransaction(lowFee); err != nil {
		t.Fatalf("SubmitTransaction lowFee failed: %v", err)
	}
	if err := bc.SubmitTransaction(midFee); err != nil {
		t.Fatalf("SubmitTransaction midFee failed: %v", err)
	}

	highFee := mustSignedTx(t, thirdAddr, senderAddr, 0, 10, 9, 1001, thirdPubHex, thirdPriv)
	if err := bc.SubmitTransaction(highFee); err != nil {
		t.Fatalf("SubmitTransaction highFee failed: %v", err)
	}

	if bc.Mempool().HasHash(lowFee.Hash) {
		t.Fatalf("expected low fee tx to be evicted")
	}
	if !bc.Mempool().HasHash(midFee.Hash) {
		t.Fatalf("expected mid fee tx to remain")
	}
	if !bc.Mempool().HasHash(highFee.Hash) {
		t.Fatalf("expected high fee tx to be admitted")
	}
}

func TestSubmitTransaction_RejectsLowFeeWhenPoolFull(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, receiverPriv, receiverPubHex := mustNewBlockchainForTest(t)

	thirdPriv, thirdPubHex, thirdAddrStr := mustKeyAndAddress(t)
	thirdAddr := pkgtypes.Address(thirdAddrStr)

	if _, err := bc.RegisterAccount(thirdAddr, thirdPubHex); err != nil {
		t.Fatalf("RegisterAccount third failed: %v", err)
	}
	if err := bc.Faucet(thirdAddr, 1000); err != nil {
		t.Fatalf("Faucet third failed: %v", err)
	}
	if err := bc.Faucet(receiverAddr, 1000); err != nil {
		t.Fatalf("Faucet receiver failed: %v", err)
	}

	bc.Mempool().SetLimits(2, 16, 8)

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 5, 1001, senderPubHex, senderPriv)
	tx2 := mustSignedTx(t, receiverAddr, senderAddr, 0, 10, 6, 1001, receiverPubHex, receiverPriv)

	if err := bc.SubmitTransaction(tx1); err != nil {
		t.Fatalf("SubmitTransaction tx1 failed: %v", err)
	}
	if err := bc.SubmitTransaction(tx2); err != nil {
		t.Fatalf("SubmitTransaction tx2 failed: %v", err)
	}

	weak := mustSignedTx(t, thirdAddr, senderAddr, 0, 10, 1, 1001, thirdPubHex, thirdPriv)
	err := bc.SubmitTransaction(weak)
	if err == nil {
		t.Fatalf("expected pool full rejection for weak fee tx")
	}
	if err.Error() != "mempool is full" {
		t.Fatalf("expected pool full error, got %v", err)
	}

	if !bc.Mempool().HasHash(tx1.Hash) || !bc.Mempool().HasHash(tx2.Hash) {
		t.Fatalf("expected original pool contents to remain unchanged")
	}
}

func TestSubmitTransaction_DoesNotEvictNonTailSenderNonce(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, receiverPriv, receiverPubHex := mustNewBlockchainForTest(t)

	thirdPriv, thirdPubHex, thirdAddrStr := mustKeyAndAddress(t)
	thirdAddr := pkgtypes.Address(thirdAddrStr)

	if _, err := bc.RegisterAccount(thirdAddr, thirdPubHex); err != nil {
		t.Fatalf("RegisterAccount third failed: %v", err)
	}
	if err := bc.Faucet(thirdAddr, 1000); err != nil {
		t.Fatalf("Faucet third failed: %v", err)
	}
	if err := bc.Faucet(receiverAddr, 1000); err != nil {
		t.Fatalf("Faucet receiver failed: %v", err)
	}

	bc.Mempool().SetLimits(3, 16, 8)

	senderTx0 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	senderTx1 := mustSignedTx(t, senderAddr, receiverAddr, 1, 10, 100, 1001, senderPubHex, senderPriv)
	otherTx := mustSignedTx(t, receiverAddr, senderAddr, 0, 10, 2, 1001, receiverPubHex, receiverPriv)

	if err := bc.SubmitTransaction(senderTx0); err != nil {
		t.Fatalf("SubmitTransaction senderTx0 failed: %v", err)
	}
	if err := bc.SubmitTransaction(senderTx1); err != nil {
		t.Fatalf("SubmitTransaction senderTx1 failed: %v", err)
	}
	if err := bc.SubmitTransaction(otherTx); err != nil {
		t.Fatalf("SubmitTransaction otherTx failed: %v", err)
	}

	incoming := mustSignedTx(t, thirdAddr, senderAddr, 0, 10, 3, 1001, thirdPubHex, thirdPriv)
	if err := bc.SubmitTransaction(incoming); err != nil {
		t.Fatalf("SubmitTransaction incoming failed: %v", err)
	}

	if bc.Mempool().HasHash(otherTx.Hash) {
		t.Fatalf("expected otherTx to be evicted as lowest evictable tail")
	}
	if !bc.Mempool().HasHash(senderTx0.Hash) {
		t.Fatalf("expected senderTx0 to remain")
	}
	if !bc.Mempool().HasHash(senderTx1.Hash) {
		t.Fatalf("expected senderTx1 to remain")
	}
	if !bc.Mempool().HasHash(incoming.Hash) {
		t.Fatalf("expected incoming tx to remain")
	}
}

func TestBlockValidate_RejectsDuplicateTransactions(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 5, 1001, senderPubHex, senderPriv)

	latest, err := bc.LatestBlock()
	if err != nil {
		t.Fatalf("LatestBlock failed: %v", err)
	}

	duplicateList := []coretypes.Transaction{*tx1, *tx1}
	b, err := block.NewBlock(
		latest.Header.Height+1,
		latest.Header.Hash,
		latest.Header.StateRoot,
		"",
		"",
		senderAddr,
		0,
		0,
		duplicateList,
		nil,
	)
	if err != nil {
		t.Fatalf("NewBlock failed: %v", err)
	}

	if err := block.Validate(b); err != block.ErrDuplicateBlockTx {
		t.Fatalf("expected ErrDuplicateBlockTx, got %v", err)
	}
}

func TestAddBlock_DoesNotValidateProposerTurnInChain(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	transaction := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	latest, err := bc.LatestBlock()
	if err != nil {
		t.Fatalf("LatestBlock failed: %v", err)
	}

	var wrong pkgtypes.Address
	for wrong == "" || wrong == senderAddr {
		_, _, addrStr := mustKeyAndAddress(t)
		wrong = pkgtypes.Address(addrStr)
	}

	b, err := block.NewBlock(
		latest.Header.Height+1,
		latest.Header.Hash,
		latest.Header.StateRoot,
		"",
		"",
		wrong,
		0,
		0,
		[]coretypes.Transaction{*transaction},
		nil,
	)
	if err != nil {
		t.Fatalf("NewBlock failed: %v", err)
	}

	if err := bc.AddBlock(b); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestBlockValidate_RejectsTamperedTxRoot(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 5, 1001, senderPubHex, senderPriv)

	latest, err := bc.LatestBlock()
	if err != nil {
		t.Fatalf("LatestBlock failed: %v", err)
	}

	b, err := block.NewBlock(
		latest.Header.Height+1,
		latest.Header.Hash,
		latest.Header.StateRoot,
		"",
		"",
		senderAddr,
		0,
		0,
		[]coretypes.Transaction{*tx1},
		nil,
	)
	if err != nil {
		t.Fatalf("NewBlock failed: %v", err)
	}

	b.Header.TxRoot = "bad_tx_root"
	hash, err := block.HeaderHash(b.Header)
	if err != nil {
		t.Fatalf("HeaderHash failed: %v", err)
	}
	b.Header.Hash = hash

	err = block.Validate(b)
	if err != block.ErrInvalidTxRoot {
		t.Fatalf("expected ErrInvalidTxRoot, got %v", err)
	}
}

func TestBlockValidate_RejectsTamperedReceiptRoot(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 5, 1001, senderPubHex, senderPriv)

	latest, err := bc.LatestBlock()
	if err != nil {
		t.Fatalf("LatestBlock failed: %v", err)
	}

	receipts := []coretypes.Receipt{
		{TxHash: tx1.Hash},
	}

	b, err := block.NewBlock(
		latest.Header.Height+1,
		latest.Header.Hash,
		latest.Header.StateRoot,
		"",
		"",
		senderAddr,
		0,
		0,
		[]coretypes.Transaction{*tx1},
		receipts,
	)
	if err != nil {
		t.Fatalf("NewBlock failed: %v", err)
	}

	b.Header.ReceiptRoot = "bad_receipt_root"
	hash, err := block.HeaderHash(b.Header)
	if err != nil {
		t.Fatalf("HeaderHash failed: %v", err)
	}
	b.Header.Hash = hash

	err = block.Validate(b)
	if err != block.ErrInvalidReceiptRoot {
		t.Fatalf("expected ErrInvalidReceiptRoot, got %v", err)
	}
}

func TestReloadBlockchain_FailsOnCorruptMetadataFile(t *testing.T) {
	dataDir := t.TempDir()

	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _, validatorSet := mustNewBlockchainForTestAtDir(t, dataDir)

	txn := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(txn); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}
	if _, err := bc.MinePending(senderAddr); err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}

	metaPath := dataDir + "\\metadata.json"
	if err := os.WriteFile(metaPath, []byte(`{"epoch":`), 0o644); err != nil {
		t.Fatalf("WriteFile corrupt metadata failed: %v", err)
	}

	_, err := NewBlockchain(dataDir, validatorSet, 1)
	if err == nil {
		t.Fatalf("expected reload to fail on corrupt metadata")
	}
}

func TestReloadBlockchain_FailsOnCorruptBlocksFile(t *testing.T) {
	dataDir := t.TempDir()

	_, _, _, _, _, _, _, validatorSet := mustNewBlockchainForTestAtDir(t, dataDir)

	blocksPath := dataDir + "\\blocks.json"
	if err := os.WriteFile(blocksPath, []byte(`[{`), 0o644); err != nil {
		t.Fatalf("WriteFile corrupt blocks failed: %v", err)
	}

	_, err := NewBlockchain(dataDir, validatorSet, 1)
	if err == nil {
		t.Fatalf("expected reload to fail on corrupt blocks file")
	}
}

func TestReloadBlockchain_IgnoresStaleTmpFiles(t *testing.T) {
	dataDir := t.TempDir()

	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _, validatorSet := mustNewBlockchainForTestAtDir(t, dataDir)

	txn := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(txn); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}
	if _, err := bc.MinePending(senderAddr); err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}

	tmpPaths := []string{
		dataDir + "\\blocks.json.tmp",
		dataDir + "\\metadata.json.tmp",
		dataDir + "\\receipts.json.tmp",
		dataDir + "\\tx_index.json.tmp",
	}

	for _, p := range tmpPaths {
		if err := os.WriteFile(p, []byte(`{"stale":true}`), 0o644); err != nil {
			t.Fatalf("WriteFile tmp failed for %s: %v", p, err)
		}
	}

	reloaded, err := NewBlockchain(dataDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("expected reload to succeed with stale tmp files, got %v", err)
	}
	if reloaded == nil {
		t.Fatalf("expected reloaded blockchain, got nil")
	}

	for _, p := range tmpPaths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("expected tmp file removed for %s, stat err=%v", p, err)
		}
	}
}

func TestSubmitTransaction_RejectsExactReplayInMempool(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	txn := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	if err := bc.SubmitTransaction(txn); err != nil {
		t.Fatalf("first SubmitTransaction failed: %v", err)
	}

	err := bc.SubmitTransaction(txn)
	if err == nil {
		t.Fatalf("expected replay rejection, got nil")
	}
	if err != ErrKnownTransaction && err != mempool.ErrDuplicateTx {
		t.Fatalf("expected ErrKnownTransaction or ErrDuplicateTx, got %v", err)
	}
}

func TestSubmitTransaction_RejectsReplayAfterMining(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	txn := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	if err := bc.SubmitTransaction(txn); err != nil {
		t.Fatalf("first SubmitTransaction failed: %v", err)
	}
	if _, err := bc.MinePending(senderAddr); err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}

	err := bc.SubmitTransaction(txn)
	if err == nil {
		t.Fatalf("expected replay rejection after mining, got nil")
	}
	if err != ErrKnownTransaction {
		t.Fatalf("expected ErrKnownTransaction, got %v", err)
	}
}

func TestSubmitTransaction_RejectsStaleNonceWhileFutureNoncePending(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustNewBlockchainForTest(t)

	tx0 := mustSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(tx0); err != nil {
		t.Fatalf("SubmitTransaction tx0 failed: %v", err)
	}

	tx1 := mustSignedTx(t, senderAddr, receiverAddr, 1, 10, 1, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(tx1); err != nil {
		t.Fatalf("SubmitTransaction tx1 failed: %v", err)
	}

	stale := mustSignedTx(t, senderAddr, receiverAddr, 0, 11, 1, 1001, senderPubHex, senderPriv)
	err := bc.SubmitTransaction(stale)
	if err == nil {
		t.Fatalf("expected stale nonce rejection, got nil")
	}
	if err != mempool.ErrDuplicateSenderNonce && err != coretypes.ErrInvalidNonce && err != ErrKnownTransaction {
		t.Fatalf("expected duplicate/invalid nonce style error, got %v", err)
	}
}
