package execution

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type engineJWTClaims struct {
	IAT int64 `json:"iat"`
}

func parseBearerToken(authHeader string) (string, error) {
	if strings.TrimSpace(authHeader) == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", fmt.Errorf("authorization header must use Bearer token")
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	if token == "" {
		return "", fmt.Errorf("missing bearer token")
	}

	return token, nil
}

func verifyJWTTokenHS256(token string, secretHex string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid jwt format")
	}

	headerSeg, payloadSeg, sigSeg := parts[0], parts[1], parts[2]

	headerRaw, err := base64.RawURLEncoding.DecodeString(headerSeg)
	if err != nil {
		return fmt.Errorf("decode jwt header failed: %w", err)
	}

	var header map[string]any
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		return fmt.Errorf("parse jwt header failed: %w", err)
	}

	alg, _ := header["alg"].(string)
	typ, _ := header["typ"].(string)

	if alg != "HS256" {
		return fmt.Errorf("unsupported jwt alg: %s", alg)
	}
	if typ != "" && typ != "JWT" {
		return fmt.Errorf("unsupported jwt typ: %s", typ)
	}

	payloadRaw, err := base64.RawURLEncoding.DecodeString(payloadSeg)
	if err != nil {
		return fmt.Errorf("decode jwt payload failed: %w", err)
	}

	var claims engineJWTClaims
	if err := json.Unmarshal(payloadRaw, &claims); err != nil {
		return fmt.Errorf("parse jwt payload failed: %w", err)
	}

	now := time.Now().Unix()
	if claims.IAT == 0 {
		return fmt.Errorf("jwt iat is required")
	}
	if claims.IAT > now+60 {
		return fmt.Errorf("jwt iat is too far in the future")
	}
	if claims.IAT < now-60 {
		return fmt.Errorf("jwt iat is too old")
	}

	secret, err := hex.DecodeString(secretHex)
	if err != nil {
		return fmt.Errorf("decode jwt secret failed: %w", err)
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(headerSeg + "." + payloadSeg))
	expectedSig := mac.Sum(nil)

	gotSig, err := base64.RawURLEncoding.DecodeString(sigSeg)
	if err != nil {
		return fmt.Errorf("decode jwt signature failed: %w", err)
	}

	if !hmac.Equal(gotSig, expectedSig) {
		return fmt.Errorf("invalid jwt signature")
	}

	return nil
}

func EngineJWTMiddleware(secretHex string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := parseBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, "engine jwt auth failed: "+err.Error(), http.StatusUnauthorized)
			return
		}

		if err := verifyJWTTokenHS256(token, secretHex); err != nil {
			http.Error(w, "engine jwt auth failed: "+err.Error(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
