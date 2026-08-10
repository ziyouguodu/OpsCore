package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"opscore/backend/internal/auth"
	"opscore/backend/internal/config"
)

func TestFailureLimiterBlocksAndResetsAttempts(t *testing.T) {
	now := time.Date(2026, 7, 10, 9, 0, 0, 0, time.UTC)
	limiter := newFailureLimiter(3, time.Minute, 5*time.Minute)
	limiter.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if allowed, _ := limiter.Allow("admin|127.0.0.1"); !allowed {
			t.Fatalf("attempt %d should still be allowed before recording failure", i+1)
		}
		limiter.Failure("admin|127.0.0.1")
	}
	if allowed, retryAfter := limiter.Allow("admin|127.0.0.1"); allowed || retryAfter != 5*time.Minute {
		t.Fatalf("expected blocked key with five minute retry, allowed=%v retry=%v", allowed, retryAfter)
	}

	now = now.Add(5*time.Minute + time.Second)
	if allowed, _ := limiter.Allow("admin|127.0.0.1"); !allowed {
		t.Fatal("key should be allowed after block expires")
	}
	limiter.Success("admin|127.0.0.1")
	if len(limiter.buckets) != 0 {
		t.Fatal("successful authentication should clear failure state")
	}
}

func TestFailureLimiterUsesIndependentKeys(t *testing.T) {
	limiter := newFailureLimiter(1, time.Minute, time.Minute)
	limiter.Failure("admin|127.0.0.1")
	if allowed, _ := limiter.Allow("admin|127.0.0.1"); allowed {
		t.Fatal("failed key should be blocked")
	}
	if allowed, _ := limiter.Allow("ops|127.0.0.1"); !allowed {
		t.Fatal("different account key should remain available")
	}
}

func TestRequestWindowLimiterCapsCostBearingRequests(t *testing.T) {
	now := time.Date(2026, 7, 10, 9, 0, 0, 0, time.UTC)
	limiter := newRequestWindowLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }
	for attempt := 1; attempt <= 2; attempt++ {
		if allowed, _ := limiter.Allow("7|127.0.0.1"); !allowed {
			t.Fatalf("request %d should be allowed", attempt)
		}
	}
	if allowed, retryAfter := limiter.Allow("7|127.0.0.1"); allowed || retryAfter != time.Minute {
		t.Fatalf("expected third request to be limited for one minute, allowed=%v retry=%v", allowed, retryAfter)
	}
	now = now.Add(time.Minute + time.Second)
	if allowed, _ := limiter.Allow("7|127.0.0.1"); !allowed {
		t.Fatal("request should be allowed after the window expires")
	}
}

func TestClientIPOnlyTrustsForwardedHeadersWhenConfigured(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.20.0.4:4567"
	req.Header.Set("X-Real-IP", "203.0.113.8")
	if got := clientIP(req, false); got != "172.20.0.4" {
		t.Fatalf("untrusted proxy header changed client IP: %s", got)
	}
	if got := clientIP(req, true); got != "203.0.113.8" {
		t.Fatalf("trusted proxy did not use validated client IP: %s", got)
	}
	req.Header.Set("X-Real-IP", "not-an-ip")
	if got := clientIP(req, true); got != "172.20.0.4" {
		t.Fatalf("invalid forwarded IP should fall back to remote address: %s", got)
	}
}

func TestLoginEndpointReturnsTooManyRequestsAfterRepeatedFailures(t *testing.T) {
	store := &mutationStore{}
	server := &Server{
		store:        store,
		signer:       auth.NewSigner("test-rate-limit-secret", time.Hour),
		cfg:          config.Config{CORSOrigin: "http://localhost:5173"},
		loginLimiter: newFailureLimiter(2, time.Minute, time.Minute),
	}
	for attempt := 1; attempt <= 3; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"wrong-password"}`))
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		server.Routes().ServeHTTP(rec, req)
		if attempt < 3 && rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d should be unauthorized, got %d", attempt, rec.Code)
		}
		if attempt == 3 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt %d should be rate limited, got %d", attempt, rec.Code)
		}
	}
}
