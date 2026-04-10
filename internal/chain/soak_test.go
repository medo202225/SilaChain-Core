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

func newSoakWallet(t *testing.T) (pkgtypes.Address, string, string) {
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

func signTransferTx(t *testing.T, privateKeyHex string, publicKeyHex string, from pkgtypes.Address, to pkgtypes.Address, value uint64, nonce uint64) coretypes.Transaction {
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

func TestProductionSoak_RepeatedTransfersAndMining(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "soak-chain")

	bc, err := NewBlockchain(dataDir, nil, 0)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}

	fromAddr, fromPriv, fromPub := newSoakWallet(t)
	toAddr, _, toPub := newSoakWallet(t)
	proposerAddr, _, proposerPub := newSoakWallet(t)

	if _, err := bc.RegisterAccount(fromAddr, fromPub); err != nil {
		t.Fatalf("register from account: %v", err)
	}
	if _, err := bc.RegisterAccount(toAddr, toPub); err != nil {
		t.Fatalf("register to account: %v", err)
	}
	if _, err := bc.RegisterAccount(proposerAddr, proposerPub); err != nil {
		t.Fatalf("register proposer account: %v", err)
	}

	if err := bc.Faucet(fromAddr, 2_000_000); err != nil {
		t.Fatalf("faucet from account: %v", err)
	}
	if err := bc.Faucet(proposerAddr, 1_000_000); err != nil {
		t.Fatalf("faucet proposer account: %v", err)
	}

	const rounds = 25

	for i := 0; i < rounds; i++ {
		transaction := signTransferTx(
			t,
			fromPriv,
			fromPub,
			fromAddr,
			toAddr,
			1,
			uint64(i),
		)

		if err := bc.SubmitTransaction(&transaction); err != nil {
			t.Fatalf("submit tx round %d: %v", i, err)
		}

		mined, err := bc.MinePending(proposerAddr)
		if err != nil {
			t.Fatalf("mine pending round %d: %v", i, err)
		}
		if mined == nil {
			t.Fatalf("expected mined block at round %d", i)
		}

		receipt, ok := bc.GetReceiptByHash(string(transaction.Hash))
		if !ok || receipt == nil {
			t.Fatalf("expected receipt for tx at round %d", i)
		}
		if !receipt.Success {
			t.Fatalf("expected success receipt at round %d, got error=%s", i, receipt.Error)
		}
	}

	height, err := bc.Height()
	if err != nil {
		t.Fatalf("height: %v", err)
	}

	if uint64(height) < rounds {
		t.Fatalf("expected height at least %d, got %d", rounds, height)
	}

	if bc.Mempool().Count() != 0 {
		t.Fatalf("expected empty mempool after soak mining, got %d", bc.Mempool().Count())
	}

	fromAcc, err := bc.GetAccount(fromAddr)
	if err != nil {
		t.Fatalf("get from account: %v", err)
	}
	if uint64(fromAcc.Nonce) != rounds {
		t.Fatalf("expected nonce %d, got %d", rounds, fromAcc.Nonce)
	}

	toAcc, err := bc.GetAccount(toAddr)
	if err != nil {
		t.Fatalf("get to account: %v", err)
	}
	if toAcc.Balance < pkgtypes.Amount(rounds) {
		t.Fatalf("expected receiver balance at least %d, got %d", rounds, toAcc.Balance)
	}
}

func TestProductionSoak_ReloadAfterSoakPreservesReceiptsAndHeight(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "soak-reload-chain")

	bc, err := NewBlockchain(dataDir, nil, 0)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}

	fromAddr, fromPriv, fromPub := newSoakWallet(t)
	toAddr, _, toPub := newSoakWallet(t)
	proposerAddr, _, proposerPub := newSoakWallet(t)

	if _, err := bc.RegisterAccount(fromAddr, fromPub); err != nil {
		t.Fatalf("register from account: %v", err)
	}
	if _, err := bc.RegisterAccount(toAddr, toPub); err != nil {
		t.Fatalf("register to account: %v", err)
	}
	if _, err := bc.RegisterAccount(proposerAddr, proposerPub); err != nil {
		t.Fatalf("register proposer account: %v", err)
	}

	if err := bc.Faucet(fromAddr, 1_000_000); err != nil {
		t.Fatalf("faucet from account: %v", err)
	}

	var lastHash pkgtypes.Hash

	for i := 0; i < 10; i++ {
		transaction := signTransferTx(
			t,
			fromPriv,
			fromPub,
			fromAddr,
			toAddr,
			1,
			uint64(i),
		)

		lastHash = transaction.Hash

		if err := bc.SubmitTransaction(&transaction); err != nil {
			t.Fatalf("submit tx round %d: %v", i, err)
		}
		if _, err := bc.MinePending(proposerAddr); err != nil {
			t.Fatalf("mine pending round %d: %v", i, err)
		}
	}

	beforeHeight, err := bc.Height()
	if err != nil {
		t.Fatalf("height before reload: %v", err)
	}

	reloaded, err := NewBlockchain(dataDir, nil, 0)
	if err != nil {
		t.Fatalf("reload blockchain: %v", err)
	}

	afterHeight, err := reloaded.Height()
	if err != nil {
		t.Fatalf("height after reload: %v", err)
	}

	if afterHeight != beforeHeight {
		t.Fatalf("expected reloaded height %d, got %d", beforeHeight, afterHeight)
	}

	receipt, ok := reloaded.GetReceiptByHash(string(lastHash))
	if !ok || receipt == nil {
		t.Fatalf("expected receipt after reload")
	}
	if !receipt.Success {
		t.Fatalf("expected successful receipt after reload, got error=%s", receipt.Error)
	}

	txObj, _, ok := reloaded.GetTransactionByHash(string(lastHash))
	if !ok || txObj == nil {
		t.Fatalf("expected transaction index entry after reload")
	}
}
