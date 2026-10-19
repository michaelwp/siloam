package auth

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var usernameRe = regexp.MustCompile(`^[a-z0-9._-]{3,32}$`)

func normalizeUsername(s string) (string, error) {
	u := strings.ToLower(strings.TrimSpace(s))
	if !usernameRe.MatchString(u) {
		return "", errors.New("username harus 3–32 karakter: huruf kecil, angka, titik, garis bawah atau tanda hubung")
	}
	return u, nil
}

func hashPassword(p string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcryptCost)
	return string(h), err
}

// dummyHash keeps sign-in timing the same whether or not the username exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equaliser"), bcryptCost)

func checkPassword(hash, p string) bool {
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(p))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)) == nil
}

// sweepEvery bounds how often the in-memory maps are scanned for expired
// entries, so a single request never pays for a full scan more than that.
const sweepEvery = time.Minute

// limiter counts events (failed sign-ins, passkey ceremonies) per key within a
// sliding window. Keys with no recent events are swept periodically.
type limiter struct {
	mu        sync.Mutex
	window    time.Duration
	hits      map[string][]time.Time
	lastSweep time.Time
}

func newLimiter(window time.Duration) *limiter {
	return &limiter{window: window, hits: map[string][]time.Time{}, lastSweep: time.Now()}
}

func (l *limiter) prune(key string, now time.Time) []time.Time {
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if now.Sub(t) < l.window {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.hits, key)
		return nil
	}
	l.hits[key] = kept
	return kept
}

func (l *limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < sweepEvery {
		return
	}
	l.lastSweep = now
	for k := range l.hits {
		l.prune(k, now)
	}
}

func (l *limiter) blocked(key string, max int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.prune(key, time.Now())) >= max
}

func (l *limiter) fail(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.sweep(now)
	for _, k := range keys {
		l.hits[k] = append(l.prune(k, now), now)
	}
}

func (l *limiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}

// maxCeremonies caps in-progress WebAuthn ceremonies held in memory.
const maxCeremonies = 10_000

// ceremonyCache holds in-progress WebAuthn ceremonies, keyed by a random
// id stored in a short-lived cookie.
type ceremonyCache struct {
	mu        sync.Mutex
	ttl       time.Duration
	max       int
	entries   map[string]ceremony
	lastSweep time.Time
}

type ceremony struct {
	data    webauthn.SessionData
	expires time.Time
}

func newCeremonyCache(ttl time.Duration) *ceremonyCache {
	return &ceremonyCache{ttl: ttl, max: maxCeremonies, entries: map[string]ceremony{}, lastSweep: time.Now()}
}

func (c *ceremonyCache) sweep(now time.Time, force bool) {
	if !force && now.Sub(c.lastSweep) < sweepEvery {
		return
	}
	c.lastSweep = now
	for k, e := range c.entries {
		if now.After(e.expires) {
			delete(c.entries, k)
		}
	}
}

// put stores a ceremony; it returns false when the cache is full of live
// ceremonies (the caller answers "busy" instead of growing memory).
func (c *ceremonyCache) put(id string, data webauthn.SessionData) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.sweep(now, false)
	if len(c.entries) >= c.max {
		c.sweep(now, true)
		if len(c.entries) >= c.max {
			return false
		}
	}
	c.entries[id] = ceremony{data: data, expires: now.Add(c.ttl)}
	return true
}

// take returns and removes the ceremony, so a challenge can be used only once.
func (c *ceremonyCache) take(id string) (webauthn.SessionData, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[id]
	delete(c.entries, id)
	if !ok || time.Now().After(e.expires) {
		return webauthn.SessionData{}, false
	}
	return e.data, true
}
