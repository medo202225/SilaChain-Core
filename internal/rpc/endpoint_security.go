package rpc

import (
	"log"
	"net"
	"net/http"
	"strings"
)

const (
	SmallJSONBodyLimit  = 1 << 12 // 4 KB
	MediumJSONBodyLimit = 1 << 15 // 32 KB
	LargeJSONBodyLimit  = 1 << 20 // 1 MB
)

func secureRead(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		applyBasicSecurityHeaders(w)

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		handler(w, r)
	}
}

func secureJSONPost(handler http.HandlerFunc, maxBodyBytes int64) http.HandlerFunc {
	return secureLimitedJSONPost(nil, handler, maxBodyBytes)
}

func secureLimitedJSONPost(limiter *RateLimiter, handler http.HandlerFunc, maxBodyBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		applyBasicSecurityHeaders(w)

		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
		if contentType == "" || !strings.HasPrefix(contentType, "application/json") {
			writeJSONError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
			return
		}

		if limiter != nil {
			if !limiter.Allow(clientIPFromRequest(r)) {
				log.Printf("security rate_limit_denied method=%s path=%s ip=%s", r.Method, r.URL.Path, clientIPFromRequest(r))
				writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
		}

		if maxBodyBytes > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}

		handler(w, r)
	}
}

func secureLocalOnlyJSONPost(limiter *RateLimiter, handler http.HandlerFunc, maxBodyBytes int64) http.HandlerFunc {
	base := secureLimitedJSONPost(limiter, handler, maxBodyBytes)

	return func(w http.ResponseWriter, r *http.Request) {
		applyBasicSecurityHeaders(w)

		if !isLocalRequest(r) {
			log.Printf("security local_only_denied method=%s path=%s ip=%s", r.Method, r.URL.Path, clientIPFromRequest(r))
			writeJSONError(w, http.StatusForbidden, "local access only")
			return
		}

		base(w, r)
	}
}

func secureWriteQuery(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		applyBasicSecurityHeaders(w)

		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		handler(w, r)
	}
}

func applyBasicSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
}

func isLocalRequest(r *http.Request) bool {
	if r == nil {
		return false
	}

	ip := strings.TrimSpace(clientIPFromRequest(r))
	if ip == "" {
		return false
	}

	if ip == "::1" || ip == "127.0.0.1" || ip == "localhost" {
		return true
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}

	return parsed.IsLoopback()
}
