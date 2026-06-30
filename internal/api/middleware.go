package api

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"BoltQ/pkg/logger"

	"github.com/gorilla/mux"
)

// isMutating reports whether a method changes state and therefore must be
// authenticated when an API key is configured.
func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// APIKeyAuth returns middleware that requires a valid API key on mutating
// requests (submit job, cancel, create/delete workflow). Read-only requests
// (GET/HEAD) stay open so the playground dashboard works without embedding a
// secret in client-side JS.
//
// If apiKey is empty the middleware is a no-op — every request passes. The
// caller is expected to log a warning at startup in that case.
//
// Accepted credentials: "X-API-Key: <key>" or "Authorization: Bearer <key>".
func APIKeyAuth(apiKey string, log *logger.Logger) mux.MiddlewareFunc {
	keyBytes := []byte(apiKey)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" || !isMutating(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			if subtle.ConstantTimeCompare([]byte(presentedKey(r)), keyBytes) != 1 {
				log.Info("Rejected unauthenticated " + r.Method + " " + r.URL.Path + " from " + clientIP(r))
				respondUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// presentedKey extracts the API key from the request, preferring X-API-Key and
// falling back to a Bearer Authorization header.
func presentedKey(r *http.Request) string {
	if k := r.Header.Get("X-API-Key"); k != "" {
		return k
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

func respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"success":false,"error":"missing or invalid API key"}`))
}

// clientIP returns the best-effort client IP, honouring X-Forwarded-For when
// the API runs behind the nginx reverse proxy.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First entry is the original client.
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimiter is a per-client-IP token-bucket limiter.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens added per second
	burst    float64 // bucket capacity
	lastSeen map[string]time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter allows `burst` requests instantly, then refills at `rps`
// requests per second per client IP.
func NewRateLimiter(rps, burst float64) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		lastSeen: make(map[string]time.Time),
		rate:     rps,
		burst:    burst,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.lastSeen[ip] = now

	b, ok := rl.buckets[ip]
	if !ok {
		rl.buckets[ip] = &bucket{tokens: rl.burst - 1, last: now}
		return true
	}

	b.tokens += now.Sub(b.last).Seconds() * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.last = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// cleanupLoop evicts buckets for IPs not seen in the last 10 minutes so the
// maps don't grow unbounded.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-10 * time.Minute)
		rl.mu.Lock()
		for ip, seen := range rl.lastSeen {
			if seen.Before(cutoff) {
				delete(rl.buckets, ip)
				delete(rl.lastSeen, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware applies the rate limiter, keyed on client IP.
func (rl *RateLimiter) Middleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.allow(clientIP(r)) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"success":false,"error":"rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
