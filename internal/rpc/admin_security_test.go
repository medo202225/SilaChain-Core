package rpc

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecureAdminJSONPost_RejectsMissingToken(t *testing.T) {
	h := secureAdminJSONPost("secret", nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, SmallJSONBodyLimit)

	req := httptest.NewRequest(http.MethodPost, "/mine", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	h(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSecureAdminJSONPost_RejectsWrongToken(t *testing.T) {
	h := secureAdminJSONPost("secret", nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, SmallJSONBodyLimit)

	req := httptest.NewRequest(http.MethodPost, "/mine", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(AdminTokenHeader, "bad")
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	h(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSecureAdminJSONPost_AllowsCorrectToken(t *testing.T) {
	h := secureAdminJSONPost("secret", nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, SmallJSONBodyLimit)

	req := httptest.NewRequest(http.MethodPost, "/mine", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(AdminTokenHeader, "secret")
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	h(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}
