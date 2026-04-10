package sdk

import (
	"testing"
	"time"
)

func TestNewWallet(t *testing.T) {
	w, err := NewWallet()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Address == "" || w.PrivateKey == "" || w.PublicKey == "" {
		t.Fatalf("expected wallet fields to be populated")
	}
}

func TestImportWalletMissingKey(t *testing.T) {
	_, err := ImportWallet("")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestWaitForReceiptRejectsEmptyServer(t *testing.T) {
	c := NewClient("http://127.0.0.1:1")
	_, err := c.WaitForReceipt("0x123", 1, 1*time.Millisecond)
	if err == nil {
		t.Fatalf("expected error")
	}
}
