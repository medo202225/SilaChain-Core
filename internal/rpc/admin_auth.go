package rpc

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const AdminTokenHeader = "X-Admin-Token"

func adminTokenFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.Header.Get(AdminTokenHeader))
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
