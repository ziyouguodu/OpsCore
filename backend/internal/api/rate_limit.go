package api

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxRateLimiterBuckets = 10_000

type failureBucket struct {
	failures     []time.Time
	blockedUntil time.Time
}

type failureLimiter struct {
	mu          sync.Mutex
	buckets     map[string]failureBucket
	max         int
	window      time.Duration
	blockFor    time.Duration
	now         func() time.Time
	nextCleanup time.Time
}

type requestWindowLimiter struct {
	mu          sync.Mutex
	buckets     map[string][]time.Time
	max         int
	window      time.Duration
	now         func() time.Time
	nextCleanup time.Time
}

func newRequestWindowLimiter(max int, window time.Duration) *requestWindowLimiter {
	return &requestWindowLimiter{buckets: make(map[string][]time.Time), max: max, window: window, now: time.Now}
}

func (l *requestWindowLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.cleanup(now)
	cutoff := now.Add(-l.window)
	values, exists := l.buckets[key]
	values = recentFailures(values, cutoff)
	if len(values) >= l.max {
		return false, max(time.Second, values[0].Add(l.window).Sub(now))
	}
	if !exists && len(l.buckets) >= maxRateLimiterBuckets {
		return false, max(time.Second, l.nextCleanup.Sub(now))
	}
	l.buckets[key] = append(values, now)
	return true, 0
}

func (l *requestWindowLimiter) cleanup(now time.Time) {
	if now.Before(l.nextCleanup) {
		return
	}
	cutoff := now.Add(-l.window)
	for key, values := range l.buckets {
		values = recentFailures(values, cutoff)
		if len(values) == 0 {
			delete(l.buckets, key)
			continue
		}
		l.buckets[key] = values
	}
	l.nextCleanup = now.Add(min(l.window, time.Minute))
}

func newFailureLimiter(max int, window, blockFor time.Duration) *failureLimiter {
	return &failureLimiter{
		buckets:  make(map[string]failureBucket),
		max:      max,
		window:   window,
		blockFor: blockFor,
		now:      time.Now,
	}
}

func (l *failureLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.cleanup(now)
	if _, exists := l.buckets[key]; !exists && len(l.buckets) >= maxRateLimiterBuckets {
		return false, max(time.Second, l.nextCleanup.Sub(now))
	}
	bucket := l.buckets[key]
	if bucket.blockedUntil.After(now) {
		return false, bucket.blockedUntil.Sub(now)
	}
	bucket.failures = recentFailures(bucket.failures, now.Add(-l.window))
	bucket.blockedUntil = time.Time{}
	l.buckets[key] = bucket
	return true, 0
}

func (l *failureLimiter) Failure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.cleanup(now)
	if _, exists := l.buckets[key]; !exists && len(l.buckets) >= maxRateLimiterBuckets {
		return
	}
	bucket := l.buckets[key]
	bucket.failures = append(recentFailures(bucket.failures, now.Add(-l.window)), now)
	if len(bucket.failures) >= l.max {
		bucket.blockedUntil = now.Add(l.blockFor)
	}
	l.buckets[key] = bucket
}

func (l *failureLimiter) cleanup(now time.Time) {
	if now.Before(l.nextCleanup) {
		return
	}
	cutoff := now.Add(-l.window)
	for key, bucket := range l.buckets {
		bucket.failures = recentFailures(bucket.failures, cutoff)
		if !bucket.blockedUntil.After(now) && len(bucket.failures) == 0 {
			delete(l.buckets, key)
			continue
		}
		l.buckets[key] = bucket
	}
	l.nextCleanup = now.Add(min(l.window, time.Minute))
}

func (l *failureLimiter) Success(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

func recentFailures(values []time.Time, cutoff time.Time) []time.Time {
	kept := values[:0]
	for _, value := range values {
		if value.After(cutoff) {
			kept = append(kept, value)
		}
	}
	return kept
}

func (s *Server) initRateLimiters() {
	s.rateLimitersOnce.Do(func() {
		if s.loginLimiter == nil {
			s.loginLimiter = newFailureLimiter(5, 15*time.Minute, 15*time.Minute)
		}
		if s.loginRequestLimiter == nil {
			s.loginRequestLimiter = newRequestWindowLimiter(60, time.Minute)
		}
		if s.credentialLimiter == nil {
			s.credentialLimiter = newFailureLimiter(5, 10*time.Minute, 10*time.Minute)
		}
		if s.copilotLimiter == nil {
			s.copilotLimiter = newRequestWindowLimiter(20, time.Minute)
		}
	})
}

func (s *Server) loginAttempts() *failureLimiter {
	s.initRateLimiters()
	return s.loginLimiter
}

func (s *Server) loginRequests() *requestWindowLimiter {
	s.initRateLimiters()
	return s.loginRequestLimiter
}

func (s *Server) credentialAttempts() *failureLimiter {
	s.initRateLimiters()
	return s.credentialLimiter
}

func (s *Server) copilotRequests() *requestWindowLimiter {
	s.initRateLimiters()
	return s.copilotLimiter
}

func loginRateKey(r *http.Request, username string, trustProxy bool) string {
	usernameHash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(username))))
	return fmt.Sprintf("%x|%s", usernameHash, clientIP(r, trustProxy))
}

func credentialRateKey(r *http.Request, userID int64, trustProxy bool) string {
	return fmt.Sprintf("%d|%s", userID, clientIP(r, trustProxy))
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if value := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(value) != nil {
			return value
		}
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			if value := strings.TrimSpace(strings.Split(forwarded, ",")[0]); net.ParseIP(value) != nil {
				return value
			}
		}
	}
	return remoteIP(r)
}

func writeRateLimit(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := max(1, int(retryAfter.Round(time.Second)/time.Second))
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeError(w, http.StatusTooManyRequests, fmt.Errorf("too many failed attempts; retry after %d seconds", seconds))
}

func writeRequestRateLimit(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := max(1, int(retryAfter.Round(time.Second)/time.Second))
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeError(w, http.StatusTooManyRequests, fmt.Errorf("request rate limit exceeded; retry after %d seconds", seconds))
}
