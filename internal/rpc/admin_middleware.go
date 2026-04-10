package rpc

import (
	"log"
	"net/http"
	"strings"
)

func secureAdminJSONPost(adminToken string, limiter *RateLimiter, handler http.HandlerFunc, maxBodyBytes int64) http.HandlerFunc {
	base := secureLocalOnlyJSONPost(limiter, handler, maxBodyBytes)

	return func(w http.ResponseWriter, r *http.Request) {
		applyBasicSecurityHeaders(w)

		expected := strings.TrimSpace(adminToken)
		if expected == "" {
			log.Printf("security admin_denied reason=token_not_configured method=%s path=%s ip=%s", r.Method, r.URL.Path, clientIPFromRequest(r))
			writeJSONError(w, http.StatusServiceUnavailable, "admin token is not configured")
			return
		}

		got := adminTokenFromRequest(r)
		if !constantTimeEqual(got, expected) {
			log.Printf("security admin_denied reason=invalid_token method=%s path=%s ip=%s", r.Method, r.URL.Path, clientIPFromRequest(r))
			writeJSONError(w, http.StatusUnauthorized, "invalid admin token")
			return
		}

		log.Printf("security admin_allowed method=%s path=%s ip=%s", r.Method, r.URL.Path, clientIPFromRequest(r))
		base(w, r)
	}
}
