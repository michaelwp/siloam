package auth

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"siloam/internal/httpx"
)

// Request rate limiting for /api, per client IP (see clientIP and
// TRUSTED_PROXIES). Two token buckets:
//
//   - every /api request uses the general bucket (APIRatePerMinute);
//   - sign-in endpoints also use a much smaller one (AuthRatePerMinute), so
//     password and passkey guessing is throttled whatever the outcome.
//
// Account lockout and the failed-sign-in counter work on top of this.

// authLimited are the endpoints that verify a secret.
var authLimited = map[string]bool{
	"/api/auth/login":                true,
	"/api/auth/passkey/login/begin":  true,
	"/api/auth/passkey/login/finish": true,
	"/api/auth/change-credentials":   true,
}

// maxBuckets caps the number of tracked clients; beyond it new clients are
// not tracked (fail open) rather than letting memory grow without bound.
const maxBuckets = 100_000

type bucket struct {
	tokens float64
	last   time.Time
}

// tokenBucket allows `burst` requests at once, refilled at `perSecond`.
type tokenBucket struct {
	mu        sync.Mutex
	perSecond float64
	burst     float64
	buckets   map[string]*bucket
	lastSweep time.Time
	full      bool // logged once when maxBuckets is reached
}

func newTokenBucket(perMinute, burst int) *tokenBucket {
	return &tokenBucket{
		perSecond: float64(perMinute) / 60,
		burst:     float64(burst),
		buckets:   map[string]*bucket{},
		lastSweep: time.Now(),
	}
}

// allow takes a token for key; when none is left it returns how long until one is.
func (tb *tokenBucket) allow(key string, now time.Time) (bool, time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.sweep(now)

	b, ok := tb.buckets[key]
	if !ok {
		if len(tb.buckets) >= maxBuckets {
			if !tb.full {
				tb.full = true
				log.Printf("rate limiter: tracking %d clients, not tracking new ones", maxBuckets)
			}
			return true, 0
		}
		b = &bucket{tokens: tb.burst, last: now}
		tb.buckets[key] = b
	}
	b.tokens = math.Min(tb.burst, b.tokens+now.Sub(b.last).Seconds()*tb.perSecond)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) / tb.perSecond * float64(time.Second))
}

// sweep drops buckets that have refilled completely (equivalent to a new client).
func (tb *tokenBucket) sweep(now time.Time) {
	if now.Sub(tb.lastSweep) < sweepEvery {
		return
	}
	tb.lastSweep = now
	refill := time.Duration(tb.burst / tb.perSecond * float64(time.Second))
	for k, b := range tb.buckets {
		if now.Sub(b.last) >= refill {
			delete(tb.buckets, k)
		}
	}
	tb.full = false
}

// RateLimit throttles /api requests per client IP. Static files are not limited.
func (s *Service) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) < 5 || r.URL.Path[:5] != "/api/" || r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		ip := s.clientIP(r)
		now := time.Now()
		if authLimited[r.URL.Path] && r.Method == http.MethodPost {
			if ok, wait := s.authRate.allow(ip, now); !ok {
				tooMany(w, wait)
				return
			}
		}
		if ok, wait := s.apiRate.allow(ip, now); !ok {
			tooMany(w, wait)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func tooMany(w http.ResponseWriter, wait time.Duration) {
	secs := int(math.Ceil(wait.Seconds()))
	w.Header().Set("Retry-After", strconv.Itoa(max(secs, 1)))
	httpx.WriteError(w, http.StatusTooManyRequests,
		"Terlalu banyak permintaan. Tunggu sebentar lalu coba lagi.", "rate_limited")
}
