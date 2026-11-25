package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestRequestWindowLimiterExpiresUnusedKeysAndCapsState(t *testing.T) {
	now := time.Date(2026, 7, 10, 9, 0, 0, 0, time.UTC)
	limiter := newRequestWindowLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }
	for i := 0; i < maxRateLimiterBuckets; i++ {
		limiter.buckets[strconv.Itoa(i)] = []time.Time{now.Add(-time.Minute)}
	}
	if allowed, _ := limiter.Allow("fresh"); !allowed || len(limiter.buckets) != 1 {
		t.Fatalf("expected expired untouched buckets to be reclaimed, allowed=%v buckets=%d", allowed, len(limiter.buckets))
	}

	full := newRequestWindowLimiter(2, time.Minute)
	full.now = func() time.Time { return now }
	for i := 0; i < maxRateLimiterBuckets; i++ {
		full.buckets[strconv.Itoa(i)] = []time.Time{now}
	}
	if allowed, retryAfter := full.Allow("overflow"); allowed || retryAfter <= 0 || len(full.buckets) != maxRateLimiterBuckets {
		t.Fatalf("expected a full limiter to reject new keys without growing, allowed=%v retry=%v buckets=%d", allowed, retryAfter, len(full.buckets))
	}
	nextCleanup := full.nextCleanup
	now = now.Add(time.Second)
	if allowed, _ := full.Allow("another-overflow"); allowed || !full.nextCleanup.Equal(nextCleanup) {
		t.Fatal("rejected fresh keys must wait for the scheduled sweep instead of triggering repeated full-map scans")
	}
}

func TestFailureLimiterExpiresUnusedKeysAndPreservesActiveLocksAtCapacity(t *testing.T) {
	now := time.Date(2026, 7, 10, 9, 0, 0, 0, time.UTC)
	limiter := newFailureLimiter(2, time.Minute, 5*time.Minute)
	limiter.now = func() time.Time { return now }
	for i := 0; i < maxRateLimiterBuckets-1; i++ {
		limiter.buckets[strconv.Itoa(i)] = failureBucket{failures: []time.Time{now.Add(-2 * time.Minute)}}
	}
	limiter.buckets["blocked"] = failureBucket{blockedUntil: now.Add(5 * time.Minute)}
	if allowed, _ := limiter.Allow("fresh"); !allowed || len(limiter.buckets) != 2 {
		t.Fatalf("expected expired keys to be swept while retaining the active lock, allowed=%v buckets=%d", allowed, len(limiter.buckets))
	}
	if allowed, retryAfter := limiter.Allow("blocked"); allowed || retryAfter != 5*time.Minute {
		t.Fatalf("expected active lock to remain enforced, allowed=%v retry=%v", allowed, retryAfter)
	}

	full := newFailureLimiter(2, time.Minute, time.Minute)
	full.now = func() time.Time { return now }
	for i := 0; i < maxRateLimiterBuckets; i++ {
		full.buckets[strconv.Itoa(i)] = failureBucket{failures: []time.Time{now}}
	}
	if allowed, retryAfter := full.Allow("overflow"); allowed || retryAfter <= 0 || len(full.buckets) != maxRateLimiterBuckets {
		t.Fatalf("expected a full limiter to reject new keys without evicting live state, allowed=%v retry=%v buckets=%d", allowed, retryAfter, len(full.buckets))
	}
	nextCleanup := full.nextCleanup
	now = now.Add(time.Second)
	if allowed, _ := full.Allow("another-overflow"); allowed || !full.nextCleanup.Equal(nextCleanup) {
		t.Fatal("rejected fresh keys must wait for the scheduled sweep instead of triggering repeated full-map scans")
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

func TestLoginEndpointDoesNotAuditRepeatedBlockedRetries(t *testing.T) {
	store := &mutationStore{}
	server := &Server{
		store:        store,
		signer:       auth.NewSigner("test-rate-limit-secret", time.Hour),
		cfg:          config.Config{CORSOrigin: "http://localhost:5173"},
		loginLimiter: newFailureLimiter(2, time.Minute, time.Minute),
	}
	for attempt := 1; attempt <= 4; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"wrong-password"}`))
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		server.Routes().ServeHTTP(rec, req)
		if attempt < 3 && rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d should be unauthorized, got %d", attempt, rec.Code)
		}
		if attempt >= 3 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt %d should be rate limited, got %d", attempt, rec.Code)
		}
	}
	if len(store.auditEvents) != 2 {
		t.Fatalf("expected only the two credential failures to be audited, got %d events", len(store.auditEvents))
	}
}

func TestLoginEndpointLimitsDistinctUsernamesPerClientIP(t *testing.T) {
	server := &Server{
		store:               &mutationStore{},
		signer:              auth.NewSigner("test-rate-limit-secret", time.Hour),
		cfg:                 config.Config{CORSOrigin: "http://localhost:5173"},
		loginLimiter:        newFailureLimiter(5, time.Minute, time.Minute),
		loginRequestLimiter: newRequestWindowLimiter(2, time.Minute),
	}
	for attempt, remoteIP := range []string{"127.0.0.1:1234", "127.0.0.1:1234", "127.0.0.1:1234", "192.0.2.4:1234", "127.0.0.1:1234"} {
		username := "user-" + strconv.Itoa(attempt+1)
		body := `{"username":"` + username + `","password":"wrong-password"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.RemoteAddr = remoteIP
		rec := httptest.NewRecorder()
		server.Routes().ServeHTTP(rec, req)
		want := http.StatusUnauthorized
		if attempt == 2 || attempt == 4 {
			want = http.StatusTooManyRequests
		}
		if rec.Code != want {
			t.Fatalf("attempt %d from %s: expected %d, got %d", attempt+1, remoteIP, want, rec.Code)
		}
	}
	if len(server.loginLimiter.buckets) != 3 {
		t.Fatalf("expected only admitted username buckets, got %d", len(server.loginLimiter.buckets))
	}
	if len(server.store.(*mutationStore).auditEvents) != 3 {
		t.Fatalf("expected only admitted credential failures to be audited, got %d events", len(server.store.(*mutationStore).auditEvents))
	}
}

func TestLoginRateKeyHashesNormalizedUsername(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	longUsername := strings.Repeat("A", maxJSONBodyBytes/2)
	key := loginRateKey(req, longUsername, false)
	if len(key) > 100 {
		t.Fatalf("rate-limit key retained an unbounded username: length=%d", len(key))
	}
	if loginRateKey(req, " Admin ", false) != loginRateKey(req, "admin", false) {
		t.Fatal("normalization should preserve the existing case and whitespace equivalence")
	}
}
