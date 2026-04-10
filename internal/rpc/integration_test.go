package rpc

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"silachain/internal/chain"
	coretypes "silachain/internal/core/types"
	"silachain/internal/validator"
	chaincrypto "silachain/pkg/crypto"
	pkgtypes "silachain/pkg/types"
)

func mustRPCKeyAndAddress(t *testing.T) (*ecdsa.PrivateKey, string, string) {
	t.Helper()

	priv, pub, err := chaincrypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	pubHex := chaincrypto.PublicKeyToHex(pub)
	if err != nil {
		t.Fatalf("PublicKeyToHex failed: %v", err)
	}

	addr := chaincrypto.PublicKeyToAddress(pub)
	if err != nil {
		t.Fatalf("PublicKeyToAddress failed: %v", err)
	}

	return priv, pubHex, string(addr)
}

func mustRPCBlockchain(t *testing.T) (*chain.Blockchain, pkgtypes.Address, *ecdsa.PrivateKey, string, pkgtypes.Address, *ecdsa.PrivateKey, string) {
	t.Helper()

	senderPriv, senderPubHex, senderAddrStr := mustRPCKeyAndAddress(t)
	receiverPriv, receiverPubHex, receiverAddrStr := mustRPCKeyAndAddress(t)

	validatorSet := validator.NewSet([]validator.Member{
		{
			Address:   pkgtypes.Address(senderAddrStr),
			PublicKey: senderPubHex,
			Power:     100,
			Stake:     1000,
		},
	})

	bc, err := chain.NewBlockchain(t.TempDir(), validatorSet, 1)
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

func mustRPCSignedTx(t *testing.T, from pkgtypes.Address, to pkgtypes.Address, nonce pkgtypes.Nonce, value pkgtypes.Amount, fee pkgtypes.Amount, chainID pkgtypes.ChainID, pubHex string, priv *ecdsa.PrivateKey) *coretypes.Transaction {
	t.Helper()

	transaction := &coretypes.Transaction{
		From:      from,
		To:        to,
		Value:     value,
		Fee:       fee,
		Nonce:     nonce,
		ChainID:   chainID,
		Timestamp: pkgtypes.Timestamp(time.Now().Unix()),
		PublicKey: pubHex,
	}

	hash, err := coretypes.ComputeHash(transaction)
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}
	transaction.Hash = hash

	if err := coretypes.SignTransaction(transaction, priv); err != nil {
		t.Fatalf("SignTransaction failed: %v", err)
	}

	return transaction
}

func decodeJSONMap(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.Unmarshal(body.Bytes(), &out); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	return out
}

func TestHealthHandler_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	HealthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestSendTxHandler_RejectsMalformedTransaction(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	body := `{"from":"bad","to":"bad","value":1,"fee":0,"nonce":1,"chain_id":1001,"timestamp":1,"public_key":"","signature":"","hash":""}`
	req := httptest.NewRequest(http.MethodPost, "/tx/send", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	SendTxHandler(bc)(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestSendTxHandler_RejectsUnknownFields(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	body := `{"from":"SILA_x","to":"SILA_y","value":1,"fee":1,"nonce":1,"chain_id":1001,"timestamp":1,"public_key":"abcdabcdabcdabcdabcd","signature":"abcdabcdabcdabcdabcd","hash":"abcdabcdabcdabcdabcd","unexpected":"boom"}`
	req := httptest.NewRequest(http.MethodPost, "/tx/send", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	SendTxHandler(bc)(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestSendTxHandler_RejectsWrongMethod(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	req := httptest.NewRequest(http.MethodGet, "/tx/send", nil)
	rr := httptest.NewRecorder()

	SendTxHandler(bc)(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestSendTxHandler_AcceptsValidTransaction(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustRPCBlockchain(t)

	transaction := mustRPCSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	payload, err := json.Marshal(transaction)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/tx/send", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	SendTxHandler(bc)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	if bc.Mempool().Count() != 1 {
		t.Fatalf("expected mempool count 1, got %d", bc.Mempool().Count())
	}
}

func TestBlockByHeightHandler_ReturnsBlock(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustRPCBlockchain(t)

	transaction := mustRPCSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(transaction); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}
	mined, err := bc.MinePending(senderAddr)
	if err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/blocks/height?height=1", nil)
	rr := httptest.NewRecorder()

	BlockByHeightHandler(bc)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	header, ok := out["header"].(map[string]any)
	if !ok {
		t.Fatalf("expected header in response")
	}
	if uint64(header["height"].(float64)) != uint64(mined.Header.Height) {
		t.Fatalf("expected height %d, got %v", mined.Header.Height, header["height"])
	}
}

func TestTxHandlers_ReturnTransactionAndReceipt(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustRPCBlockchain(t)

	transaction := mustRPCSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	if err := bc.SubmitTransaction(transaction); err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}
	if _, err := bc.MinePending(senderAddr); err != nil {
		t.Fatalf("MinePending failed: %v", err)
	}

	reqTx := httptest.NewRequest(http.MethodGet, "/tx/by-hash?hash="+string(transaction.Hash), nil)
	rrTx := httptest.NewRecorder()
	TxByHashHandler(bc)(rrTx, reqTx)

	if rrTx.Code != http.StatusOK {
		t.Fatalf("expected tx handler 200, got %d body=%s", rrTx.Code, rrTx.Body.String())
	}

	reqReceipt := httptest.NewRequest(http.MethodGet, "/tx/receipt?hash="+string(transaction.Hash), nil)
	rrReceipt := httptest.NewRecorder()
	TxReceiptHandler(bc)(rrReceipt, reqReceipt)

	if rrReceipt.Code != http.StatusOK {
		t.Fatalf("expected receipt handler 200, got %d body=%s", rrReceipt.Code, rrReceipt.Body.String())
	}
}

func TestGlobalRateLimit_HealthEndpoint(t *testing.T) {
	resetRateLimitersForTest()
	lightRateLimiter = NewRateLimiter(LightRateWindow, 5)
	defer resetRateLimitersForTest()

	var lastCode int
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()

		secureWithPolicy(HealthHandler, lightPolicy())(rr, req)
		lastCode = rr.Code
	}

	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("expected final status 429, got %d", lastCode)
	}
}

func TestSecureJSONPost_RejectsMissingContentType(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	body := bytes.NewBufferString(`{"address":"SILA_123","public_key":"abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/accounts/register", body)
	rr := httptest.NewRecorder()

	secureJSONPost(RegisterAccountHandler(bc), SmallJSONBodyLimit)(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSecureJSONPost_RejectsWrongContentType(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	body := bytes.NewBufferString(`{"address":"SILA_123","public_key":"abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/accounts/register", body)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	secureJSONPost(RegisterAccountHandler(bc), SmallJSONBodyLimit)(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSecureJSONPost_RejectsOversizedBody(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	oversized := strings.Repeat("a", 5000)
	body := bytes.NewBufferString(oversized)
	req := httptest.NewRequest(http.MethodPost, "/accounts/register", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	secureJSONPost(RegisterAccountHandler(bc), 32)(rr, req)

	if rr.Code == http.StatusOK {
		t.Fatalf("expected non-200 for oversized body, got %d", rr.Code)
	}
}

func TestSecureWriteQuery_RejectsWrongMethod(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	req := httptest.NewRequest(http.MethodPost, "/blocks/height?height=0", nil)
	rr := httptest.NewRecorder()

	secureWriteQuery(BlockByHeightHandler(bc))(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSecureRead_AddsSecurityHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	secureRead(HealthHandler)(rr, req)

	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff header")
	}
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing frame options header")
	}
	if rr.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("missing cache-control header")
	}
}

func TestSecureLimitedJSONPost_RateLimitsByEndpoint(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	limiter := NewRateLimiter(time.Minute, 1)

	body1 := bytes.NewBufferString(`{"address":"SILA_test","amount":1}`)
	req1 := httptest.NewRequest(http.MethodPost, "/faucet", body1)
	req1.Header.Set("Content-Type", "application/json")
	req1.RemoteAddr = "127.0.0.1:12345"
	rr1 := httptest.NewRecorder()

	secureLimitedJSONPost(limiter, FaucetHandler(bc), SmallJSONBodyLimit)(rr1, req1)

	body2 := bytes.NewBufferString(`{"address":"SILA_test","amount":1}`)
	req2 := httptest.NewRequest(http.MethodPost, "/faucet", body2)
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = "127.0.0.1:12345"
	rr2 := httptest.NewRecorder()

	secureLimitedJSONPost(limiter, FaucetHandler(bc), SmallJSONBodyLimit)(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestSecureLimitedJSONPost_DifferentIPsDoNotShareLimit(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	limiter := NewRateLimiter(time.Minute, 1)

	body1 := bytes.NewBufferString(`{"address":"SILA_test","amount":1}`)
	req1 := httptest.NewRequest(http.MethodPost, "/faucet", body1)
	req1.Header.Set("Content-Type", "application/json")
	req1.RemoteAddr = "127.0.0.1:11111"
	rr1 := httptest.NewRecorder()

	secureLimitedJSONPost(limiter, FaucetHandler(bc), SmallJSONBodyLimit)(rr1, req1)

	body2 := bytes.NewBufferString(`{"address":"SILA_test","amount":1}`)
	req2 := httptest.NewRequest(http.MethodPost, "/faucet", body2)
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = "127.0.0.2:22222"
	rr2 := httptest.NewRecorder()

	secureLimitedJSONPost(limiter, FaucetHandler(bc), SmallJSONBodyLimit)(rr2, req2)

	if rr2.Code == http.StatusTooManyRequests {
		t.Fatalf("different IPs should not share limiter state")
	}
}

func TestSecureLimitedJSONPost_MethodStillEnforcedBeforeHandler(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	limiter := NewRateLimiter(time.Minute, 1)

	req := httptest.NewRequest(http.MethodGet, "/mine", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	secureLimitedJSONPost(limiter, MineHandler(bc), SmallJSONBodyLimit)(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSecureLocalOnlyJSONPost_RejectsRemoteIP(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	body := bytes.NewBufferString(`{"proposer":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/mine", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	rr := httptest.NewRecorder()

	secureLocalOnlyJSONPost(nil, MineHandler(bc), SmallJSONBodyLimit)(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSecureLocalOnlyJSONPost_AllowsLoopbackIPv4(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	body := bytes.NewBufferString(`{"proposer":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/mine", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	secureLocalOnlyJSONPost(nil, MineHandler(bc), SmallJSONBodyLimit)(rr, req)

	if rr.Code == http.StatusForbidden {
		t.Fatalf("expected non-403 for loopback IPv4, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSecureLocalOnlyJSONPost_AllowsLoopbackIPv6(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	body := bytes.NewBufferString(`{"proposer":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/mine", body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "[::1]:12345"
	rr := httptest.NewRecorder()

	secureLocalOnlyJSONPost(nil, MineHandler(bc), SmallJSONBodyLimit)(rr, req)

	if rr.Code == http.StatusForbidden {
		t.Fatalf("expected non-403 for loopback IPv6, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestIsLocalRequest_UsesForwardedHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/mine", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.Header.Set("X-Forwarded-For", "127.0.0.1")

	if !isLocalRequest(req) {
		t.Fatalf("expected forwarded loopback IP to be treated as local")
	}
}

func TestSecureAdminJSONPost_UsesLocalOnlyFallbackPolicyExternally(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	req := httptest.NewRequest(http.MethodPost, "/mine", bytes.NewBufferString(`{"proposer":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	rr := httptest.NewRecorder()

	secureLocalOnlyJSONPost(nil, MineHandler(bc), SmallJSONBodyLimit)(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSecureAdminJSONPost_RejectsRemoteEvenWithCorrectToken(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	req := httptest.NewRequest(http.MethodPost, "/mine", bytes.NewBufferString(`{"proposer":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(AdminTokenHeader, "secret")
	req.RemoteAddr = "8.8.8.8:12345"
	rr := httptest.NewRecorder()

	secureAdminJSONPost("secret", nil, MineHandler(bc), SmallJSONBodyLimit)(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSendTxHandler_RejectsMalformedJSONSyntax(t *testing.T) {
	bc, _, _, _, _, _, _ := mustRPCBlockchain(t)

	body := bytes.NewBufferString(`{"from":"bad",`)
	req := httptest.NewRequest(http.MethodPost, "/tx/send", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	secureLimitedJSONPost(NewRateLimiter(time.Minute, 10), SendTxHandler(bc), MediumJSONBodyLimit)(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSendTxHandler_RejectsExactReplay(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustRPCBlockchain(t)

	transaction := mustRPCSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	rawBody, err := json.Marshal(transaction)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	req1 := httptest.NewRequest(http.MethodPost, "/tx/send", bytes.NewReader(rawBody))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()

	secureLimitedJSONPost(NewRateLimiter(time.Minute, 10), SendTxHandler(bc), MediumJSONBodyLimit)(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first submit 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/tx/send", bytes.NewReader(rawBody))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()

	secureLimitedJSONPost(NewRateLimiter(time.Minute, 10), SendTxHandler(bc), MediumJSONBodyLimit)(rr2, req2)

	if rr2.Code == http.StatusOK {
		t.Fatalf("expected replay rejection, got 200 body=%s", rr2.Body.String())
	}
}

func TestSendTxHandler_RejectsDuplicateSenderNonceViaRPC(t *testing.T) {
	bc, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _ := mustRPCBlockchain(t)

	tx1 := mustRPCSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	tx2 := mustRPCSignedTx(t, senderAddr, receiverAddr, 0, 11, 1, 1001, senderPubHex, senderPriv)

	raw1, err := json.Marshal(tx1)
	if err != nil {
		t.Fatalf("json.Marshal tx1 failed: %v", err)
	}
	raw2, err := json.Marshal(tx2)
	if err != nil {
		t.Fatalf("json.Marshal tx2 failed: %v", err)
	}

	req1 := httptest.NewRequest(http.MethodPost, "/tx/send", bytes.NewReader(raw1))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	secureLimitedJSONPost(NewRateLimiter(time.Minute, 10), SendTxHandler(bc), MediumJSONBodyLimit)(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first submit 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/tx/send", bytes.NewReader(raw2))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	secureLimitedJSONPost(NewRateLimiter(time.Minute, 10), SendTxHandler(bc), MediumJSONBodyLimit)(rr2, req2)

	if rr2.Code == http.StatusOK {
		t.Fatalf("expected duplicate sender nonce rejection, got 200 body=%s", rr2.Body.String())
	}
}
