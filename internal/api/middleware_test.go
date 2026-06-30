package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"BoltQ/pkg/logger"
)

func TestAPIKeyAuth(t *testing.T) {
	log := logger.NewLogger("test")
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	cases := []struct {
		name       string
		apiKey     string
		method     string
		header     map[string]string
		wantStatus int
	}{
		{"no key configured allows mutating", "", http.MethodPost, nil, http.StatusOK},
		{"GET never requires key", "secret", http.MethodGet, nil, http.StatusOK},
		{"POST without key rejected", "secret", http.MethodPost, nil, http.StatusUnauthorized},
		{"POST wrong key rejected", "secret", http.MethodPost, map[string]string{"X-API-Key": "nope"}, http.StatusUnauthorized},
		{"POST correct X-API-Key allowed", "secret", http.MethodPost, map[string]string{"X-API-Key": "secret"}, http.StatusOK},
		{"POST correct Bearer allowed", "secret", http.MethodPost, map[string]string{"Authorization": "Bearer secret"}, http.StatusOK},
		{"DELETE without key rejected", "secret", http.MethodDelete, nil, http.StatusUnauthorized},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := APIKeyAuth(c.apiKey, log)(ok)
			req := httptest.NewRequest(c.method, "/api/v1/jobs", nil)
			for k, v := range c.header {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != c.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, c.wantStatus)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	// burst of 3, refill effectively 0 within the test window
	rl := NewRateLimiter(0.0001, 3)
	h := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func() int {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "203.0.113.7:5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	for i := 0; i < 3; i++ {
		if got := call(); got != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i+1, got)
		}
	}
	if got := call(); got != http.StatusTooManyRequests {
		t.Errorf("4th request: status = %d, want 429", got)
	}
}
