package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizeUsername(t *testing.T) {
	ok := []struct{ in, want string }{{"  Admin\t", "admin"}, {"staf.1", "staf.1"}, {"a-b_c", "a-b_c"}}
	for _, c := range ok {
		if got, err := normalizeUsername(c.in); err != nil || got != c.want {
			t.Errorf("normalizeUsername(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	for _, in := range []string{"", "ab", "with space", "tanda!", "x123456789012345678901234567890123"} {
		if _, err := normalizeUsername(in); err == nil {
			t.Errorf("normalizeUsername(%q) accepted", in)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	good := []string{"Kolom9-Paal2!", "Tetap Semangat 7?", "Ñandú#2027abc"}
	for _, p := range good {
		if err := validatePassword(p, "staf1"); err != nil {
			t.Errorf("validatePassword(%q) = %v", p, err)
		}
	}
	bad := map[string]string{
		"Ab1!":                "minimal 10 karakter",
		"alllowercase1!":      "huruf besar dan huruf kecil",
		"NoDigitsHere!!":      "angka",
		"NoSymbols12345":      "simbol",
		"Staf1-is-me-2027":    "username",
		"Password123!":        "umum",
		"J3maat!!JEMAAT2027#": "", // "jmaatjemaat" is not a common word → accepted
	}
	for p, want := range bad {
		err := validatePassword(p, "staf1")
		if want == "" {
			if err != nil {
				t.Errorf("validatePassword(%q) = %v, want ok", p, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("validatePassword(%q) = %v, want error mentioning %q", p, err, want)
		}
	}
	if validatePassword(strings.Repeat("Aa1!", 19), "") == nil {
		t.Error("76-byte password accepted")
	}
}

func TestCheckPassword(t *testing.T) {
	h, err := hashPassword("rahasia-123")
	if err != nil {
		t.Fatal(err)
	}
	if !checkPassword(h, "rahasia-123") || checkPassword(h, "salah") || checkPassword("", "rahasia-123") {
		t.Error("checkPassword gave wrong result")
	}
}

func TestLimiter(t *testing.T) {
	l := newLimiter(time.Minute)
	for range 4 {
		l.fail("k")
	}
	if l.blocked("k", 5) {
		t.Error("blocked after 4 failures")
	}
	l.fail("k")
	if !l.blocked("k", 5) {
		t.Error("not blocked after 5 failures")
	}
	l.reset("k")
	if l.blocked("k", 5) {
		t.Error("still blocked after reset")
	}
}

func TestGeneratePasswordMeetsRules(t *testing.T) {
	for range 500 {
		p := generatePassword(14)
		if err := validatePassword(p, "admin"); err != nil || len(p) != 14 {
			t.Fatalf("generated %q: %v", p, err)
		}
	}
}

func TestClientIPTrustedProxies(t *testing.T) {
	proxies, err := ParseTrustedProxies("10.0.0.0/8, 192.168.1.5")
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{cfg: Config{TrustedProxies: proxies}}
	cases := []struct{ remote, xff, want string }{
		{"203.0.113.9:5000", "1.2.3.4", "203.0.113.9"},                       // untrusted peer: header ignored
		{"10.0.0.2:5000", "198.51.100.7", "198.51.100.7"},                    // via trusted proxy
		{"10.0.0.2:5000", "6.6.6.6, 198.51.100.7, 10.1.1.1", "198.51.100.7"}, // spoofed left entry ignored
		{"192.168.1.5:80", "", "192.168.1.5"},                                // no header
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = c.remote
		if c.xff != "" {
			r.Header.Set("X-Forwarded-For", c.xff)
		}
		if got := s.clientIP(r); got != c.want {
			t.Errorf("clientIP(%s, %q) = %s, want %s", c.remote, c.xff, got, c.want)
		}
	}
	if _, err := ParseTrustedProxies("not-an-ip"); err == nil {
		t.Error("bad proxy accepted")
	}
}

func TestLimiterSweepsIdleKeys(t *testing.T) {
	l := newLimiter(time.Millisecond)
	l.fail("u:random-1", "u:random-2")
	time.Sleep(5 * time.Millisecond)
	l.lastSweep = time.Time{} // force the next sweep
	l.fail("other")
	if len(l.hits) != 1 {
		t.Fatalf("idle keys kept: %v", l.hits)
	}
}

func TestTokenBucket(t *testing.T) {
	tb := newTokenBucket(60, 3) // 1/s, burst 3
	now := time.Now()
	for i := range 3 {
		if ok, _ := tb.allow("ip", now); !ok {
			t.Fatalf("request %d rejected within burst", i+1)
		}
	}
	ok, wait := tb.allow("ip", now)
	if ok || wait <= 0 || wait > time.Second {
		t.Fatalf("4th request: ok=%v wait=%v", ok, wait)
	}
	if ok, _ := tb.allow("other-ip", now); !ok {
		t.Fatal("other client affected")
	}
	if ok, _ := tb.allow("ip", now.Add(time.Second)); !ok {
		t.Fatal("not refilled after 1s")
	}
	tb.lastSweep = time.Time{}
	tb.allow("x", now.Add(time.Hour)) // sweeps refilled buckets
	if len(tb.buckets) != 1 {
		t.Fatalf("idle buckets kept: %d", len(tb.buckets))
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	s := &Service{apiRate: newTokenBucket(600, 5), authRate: newTokenBucket(20, 2)}
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := s.RateLimit(ok)
	do := func(method, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r.RemoteAddr = "203.0.113.9:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}
	// sign-in: burst 2, then 429 with Retry-After
	codes := []int{do("POST", "/api/auth/login").Code, do("POST", "/api/auth/login").Code}
	rec := do("POST", "/api/auth/login")
	if codes[0] != 204 || codes[1] != 204 || rec.Code != 429 || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("auth limit: %v then %d retry=%q", codes, rec.Code, rec.Header().Get("Retry-After"))
	}
	// general API: 5 burst total (2 already used by the logins above)
	for range 3 {
		if c := do("GET", "/api/families").Code; c != 204 {
			t.Fatalf("api request rejected early: %d", c)
		}
	}
	if c := do("GET", "/api/members").Code; c != 429 {
		t.Fatalf("api flood not limited: %d", c)
	}
	// static files and health are never limited
	for _, p := range []string{"/", "/assets/app.js", "/api/health"} {
		if c := do("GET", p).Code; c != 204 {
			t.Fatalf("%s limited: %d", p, c)
		}
	}
}

func TestClientIPHeader(t *testing.T) {
	s := &Service{cfg: Config{ClientIPHeader: "Fly-Client-IP"}}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "[fdaa::1]:443" // Fly's proxy
	r.Header.Set("Fly-Client-IP", "198.51.100.7")
	if got := s.clientIP(r); got != "198.51.100.7" {
		t.Fatalf("clientIP = %s", got)
	}
	r.Header.Set("Fly-Client-IP", "garbage")
	if got := s.clientIP(r); got != "fdaa::1" {
		t.Fatalf("invalid header should fall back to peer, got %s", got)
	}
}

func TestPublicReadOnlyData(t *testing.T) {
	cases := []struct {
		method, path string
		public       bool
	}{
		{"GET", "/api/families", true},
		{"GET", "/api/families/abc", true},
		{"GET", "/api/members", true},
		{"HEAD", "/api/members/abc", true},
		{"POST", "/api/families", false},
		{"PUT", "/api/members/abc", false},
		{"DELETE", "/api/families/abc", false},
		{"GET", "/api/users", false},
		{"GET", "/api/auth/me", false},
		{"GET", "/api/familiesX", false},
		{"GET", "/api/app/android", true},
		{"GET", "/api/admin/android", false},
		{"GET", "/api/admin/audit", false},
		{"POST", "/api/app/android", false},
	}
	for _, c := range cases {
		if got := publicRead(httptest.NewRequest(c.method, c.path, nil)); got != c.public {
			t.Errorf("%s %s public=%v, want %v", c.method, c.path, got, c.public)
		}
	}
}
