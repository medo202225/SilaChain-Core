package rpc

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBodyLimitBytes = int64(1 << 20) // 1 MiB

	LightBodyLimitBytes    = int64(16 * 1024)
	StandardBodyLimitBytes = int64(256 * 1024)
	HeavyBodyLimitBytes    = int64(64 * 1024)

	LightRateWindow    = 10 * time.Second
	StandardRateWindow = 10 * time.Second
	HeavyRateWindow    = 10 * time.Second

	LightRateMaxHits    = 120
	StandardRateMaxHits = 60
	HeavyRateMaxHits    = 20
)

type rateEntry struct {
	Count     int
	ExpiresAt time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	maxHits int
	items   map[string]rateEntry
}

func NewRateLimiter(window time.Duration, maxHits int) *RateLimiter {
	return &RateLimiter{
		window:  window,
		maxHits: maxHits,
		items:   map[string]rateEntry{},
	}
}

func (r *RateLimiter) Allow(key string) bool {
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	for k, v := range r.items {
		if now.After(v.ExpiresAt) {
			delete(r.items, k)
		}
	}

	entry, ok := r.items[key]
	if !ok || now.After(entry.ExpiresAt) {
		r.items[key] = rateEntry{
			Count:     1,
			ExpiresAt: now.Add(r.window),
		}
		return true
	}

	if entry.Count >= r.maxHits {
		return false
	}

	entry.Count++
	r.items[key] = entry
	return true
}

var (
	lightRateLimiter    = NewRateLimiter(LightRateWindow, LightRateMaxHits)
	standardRateLimiter = NewRateLimiter(StandardRateWindow, StandardRateMaxHits)
	heavyRateLimiter    = NewRateLimiter(HeavyRateWindow, HeavyRateMaxHits)
)

type EndpointPolicy struct {
	BodyLimit int64
	Limiter   *RateLimiter
}

func lightPolicy() EndpointPolicy {
	return EndpointPolicy{
		BodyLimit: LightBodyLimitBytes,
		Limiter:   lightRateLimiter,
	}
}

func standardPolicy() EndpointPolicy {
	return EndpointPolicy{
		BodyLimit: StandardBodyLimitBytes,
		Limiter:   standardRateLimiter,
	}
}

func heavyPolicy() EndpointPolicy {
	return EndpointPolicy{
		BodyLimit: HeavyBodyLimitBytes,
		Limiter:   heavyRateLimiter,
	}
}

func resetRateLimitersForTest() {
	lightRateLimiter = NewRateLimiter(LightRateWindow, LightRateMaxHits)
	standardRateLimiter = NewRateLimiter(StandardRateWindow, StandardRateMaxHits)
	heavyRateLimiter = NewRateLimiter(HeavyRateWindow, HeavyRateMaxHits)
}

func clientIP(r *http.Request) string {
	if r == nil {
		return "unknown"
	}

	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}

	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}

	return "unknown"
}

func limitBody(next http.HandlerFunc, maxBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.Body != nil && maxBytes > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next(w, r)
	}
}

func rateLimit(next http.HandlerFunc, limiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if limiter == nil {
			next(w, r)
			return
		}

		key := clientIP(r) + "|" + r.Method + "|" + r.URL.Path
		if !limiter.Allow(key) {
			writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next(w, r)
	}
}

func secureWithPolicy(next http.HandlerFunc, policy EndpointPolicy) http.HandlerFunc {
	return rateLimit(limitBody(next, policy.BodyLimit), policy.Limiter)
}

func secureJSON(next http.HandlerFunc, maxBytes int64) http.HandlerFunc {
	return rateLimit(limitBody(next, maxBytes), standardRateLimiter)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error":   true,
		"status":  status,
		"message": message,
	})
}
