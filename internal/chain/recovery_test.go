package chain

import (
	"path/filepath"
	"testing"
	"time"

	coretypes "silachain/internal/core/types"
	"silachain/internal/protocol"
	chaincrypto "silachain/pkg/crypto"
	pkgtypes "silachain/pkg/types"
)

func newRecoveryWallet(t *testing.T) (pkgtypes.Address, string, string) {
	t.Helper()

	priv, pub, err := chaincrypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}

	address := chaincrypto.PublicKeyToAddress(pub)
	privateKeyHex := chaincrypto.PrivateKeyToHex(priv)
	publicKeyHex := chaincrypto.PublicKeyToHex(pub)

	return pkgtypes.Address(address), privateKeyHex, publicKeyHex
}

func signRecoveryTransferTx(t *testing.T, privateKeyHex string, publicKeyHex string, from pkgtypes.Address, to pkgtypes.Address, value uint64, nonce uint64) coretypes.Transaction {
	t.Helper()

	transaction := coretypes.Transaction{
		Type:      coretypes.TypeTransfer,
		From:      from,
		To:        to,
		Value:     pkgtypes.Amount(value),
		Fee:       0,
		GasPrice:  1,
		GasLimit:  21000,
		Nonce:     pkgtypes.Nonce(nonce),
		ChainID:   pkgtypes.ChainID(protocol.DefaultMainnetParams().ChainID),
		Timestamp: pkgtypes.Timestamp(time.Now().Unix()),
		PublicKey: publicKeyHex,
	}

	hashHex, err := chaincrypto.HashJSON(transaction.SigningPayload())
	if err != nil {
		t.Fatalf("hash signing payload: %v", err)
	}

	priv, err := chaincrypto.HexToPrivateKey(privateKeyHex)
	if err != nil {
		t.Fatalf("hex to private key: %v", err)
	}

	sigHex, err := chaincrypto.SignHashHex(priv, hashHex)
	if err != nil {
		t.Fatalf("sign hash: %v", err)
	}

	transaction.Hash = pkgtypes.Hash(hashHex)
	transaction.Signature = sigHex
	return transaction
}

func TestRestartRecovery_MultiRestartPreservesHeightAndBalances(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "recovery-multi-restart")

	bc, err := NewBlockchain(dataDir, nil, 0)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}

	fromAddr, fromPriv, fromPub := newRecoveryWallet(t)
	toAddr, _, toPub := newRecoveryWallet(t)
	proposerAddr, _, proposerPub := newRecoveryWallet(t)

	if _, err := bc.RegisterAccount(fromAddr, fromPub); err != nil {
		t.Fatalf("register from: %v", err)
	}
	if _, err := bc.RegisterAccount(toAddr, toPub); err != nil {
		t.Fatalf("register to: %v", err)
	}
	if _, err := bc.RegisterAccount(proposerAddr, proposerPub); err != nil {
		t.Fatalf("register proposer: %v", err)
	}

	if err := bc.Faucet(fromAddr, 1_000_000); err != nil {
		t.Fatalf("faucet from: %v", err)
	}

	const rounds = 6
	expectedReceived := uint64(0)

	for i := 0; i < rounds; i++ {
		txObj := signRecoveryTransferTx(t, fromPriv, fromPub, fromAddr, toAddr, 3, uint64(i))

		if err := bc.SubmitTransaction(&txObj); err != nil {
			t.Fatalf("submit tx round %d: %v", i, err)
		}
		if _, err := bc.MinePending(proposerAddr); err != nil {
			t.Fatalf("mine tx round %d: %v", i, err)
		}

		expectedReceived += 3

		reloaded, err := NewBlockchain(dataDir, nil, 0)
		if err != nil {
			t.Fatalf("reload round %d: %v", i, err)
		}

		height, err := reloaded.Height()
		if err != nil {
			t.Fatalf("height round %d: %v", i, err)
		}
		if uint64(height) < uint64(i+1) {
			t.Fatalf("expected height at least %d after restart, got %d", i+1, height)
		}

		receipt, ok := reloaded.GetReceiptByHash(string(txObj.Hash))
		if !ok || receipt == nil {
			t.Fatalf("missing receipt after restart round %d", i)
		}
		if !receipt.Success {
			t.Fatalf("expected successful receipt after restart round %d, got error=%s", i, receipt.Error)
		}

		toAcc, err := reloaded.GetAccount(toAddr)
		if err != nil {
			t.Fatalf("get receiver after restart round %d: %v", i, err)
		}
		if uint64(toAcc.Balance) < expectedReceived {
			t.Fatalf("expected receiver balance at least %d, got %d", expectedReceived, toAcc.Balance)
		}

		bc = reloaded
	}
}

func TestRestartRecovery_ContractStateAndCodePersist(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "recovery-contract-state")

	bc, err := NewBlockchain(dataDir, nil, 0)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}

	ownerAddr, _, ownerPub := newRecoveryWallet(t)
	contractAddr := pkgtypes.Address("SILA_CONTRACT_RECOVERY_001")

	if _, err := bc.RegisterAccount(ownerAddr, ownerPub); err != nil {
		t.Fatalf("register owner: %v", err)
	}

	acc, codeHash, err := bc.DeployContract(contractAddr, "6001600055", 0)
	if err != nil {
		t.Fatalf("deploy contract: %v", err)
	}
	if acc == nil {
		t.Fatalf("expected contract account")
	}
	if codeHash == "" {
		t.Fatalf("expected code hash")
	}

	if err := bc.SetContractStorage(contractAddr, "greeting", "hello"); err != nil {
		t.Fatalf("set storage: %v", err)
	}

	reloaded, err := NewBlockchain(dataDir, nil, 0)
	if err != nil {
		t.Fatalf("reload blockchain: %v", err)
	}

	code, ok := reloaded.GetContractCode(contractAddr)
	if !ok {
		t.Fatalf("expected contract code after reload")
	}
	if code != "6001600055" {
		t.Fatalf("expected code 6001600055, got %s", code)
	}

	value, ok := reloaded.GetContractStorage(contractAddr, "greeting")
	if !ok {
		t.Fatalf("expected storage key after reload")
	}
	if value != "hello" {
		t.Fatalf("expected storage value hello, got %s", value)
	}

	root, err := reloaded.GetContractStorageRoot(contractAddr)
	if err != nil {
		t.Fatalf("storage root after reload: %v", err)
	}
	if root == "" {
		t.Fatalf("expected non-empty storage root after reload")
	}
}

func TestRestartRecovery_TransactionIndexAndReceiptAvailability(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "recovery-index-receipt")

	bc, err := NewBlockchain(dataDir, nil, 0)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}

	fromAddr, fromPriv, fromPub := newRecoveryWallet(t)
	toAddr, _, toPub := newRecoveryWallet(t)
	proposerAddr, _, proposerPub := newRecoveryWallet(t)

	if _, err := bc.RegisterAccount(fromAddr, fromPub); err != nil {
		t.Fatalf("register from: %v", err)
	}
	if _, err := bc.RegisterAccount(toAddr, toPub); err != nil {
		t.Fatalf("register to: %v", err)
	}
	if _, err := bc.RegisterAccount(proposerAddr, proposerPub); err != nil {
		t.Fatalf("register proposer: %v", err)
	}

	if err := bc.Faucet(fromAddr, 500_000); err != nil {
		t.Fatalf("faucet from: %v", err)
	}

	txObj := signRecoveryTransferTx(t, fromPriv, fromPub, fromAddr, toAddr, 7, 0)

	if err := bc.SubmitTransaction(&txObj); err != nil {
		t.Fatalf("submit tx: %v", err)
	}
	if _, err := bc.MinePending(proposerAddr); err != nil {
		t.Fatalf("mine tx: %v", err)
	}

	reloaded, err := NewBlockchain(dataDir, nil, 0)
	if err != nil {
		t.Fatalf("reload blockchain: %v", err)
	}

	foundTx, height, ok := reloaded.GetTransactionByHash(string(txObj.Hash))
	if !ok || foundTx == nil {
		t.Fatalf("expected tx by hash after reload")
	}
	if height == 0 {
		t.Fatalf("expected non-zero block height for tx after reload")
	}

	receipt, ok := reloaded.GetReceiptByHash(string(txObj.Hash))
	if !ok || receipt == nil {
		t.Fatalf("expected receipt by hash after reload")
	}
	if !receipt.Success {
		t.Fatalf("expected successful receipt after reload, got error=%s", receipt.Error)
	}

	latest, err := reloaded.LatestBlock()
	if err != nil {
		t.Fatalf("latest block: %v", err)
	}
	if latest == nil || len(latest.Receipts) == 0 {
		t.Fatalf("expected latest block receipts after reload")
	}
}
