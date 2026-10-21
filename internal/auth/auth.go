// Package auth provides sign-in (password and passkey), sessions, the
// first-run admin account and user management.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/httpx"
	"siloam/internal/model"
	"siloam/internal/store"
)

const (
	sessionCookie  = "siloam_session"
	ceremonyCookie = "siloam_webauthn"
)

type Config struct {
	RPID                 string   // WebAuthn relying party ID, e.g. "localhost" or "jemaat.example.org"
	RPName               string   // shown by the authenticator
	Origins              []string // allowed WebAuthn origins, e.g. "http://localhost:8080"
	SecureCookie         bool     // set the Secure flag (enable behind HTTPS)
	SessionTTL           time.Duration
	MaxFailedLogins      int            // wrong passwords in a row before the account locks (default 3)
	LockDuration         time.Duration  // how long a lock lasts (default 3 minutes)
	APIRatePerMinute     int            // /api requests per client IP per minute (default 600)
	AuthRatePerMinute    int            // sign-in attempts per client IP per minute (default 20)
	TrustedProxies       []netip.Prefix // reverse proxies whose X-Forwarded-For is believed
	ClientIPHeader       string         // header set by the platform's edge proxy, e.g. "Fly-Client-IP"
	InitialAdminUser     string
	InitialAdminPassword string
}

type Service struct {
	cfg      Config
	store    *store.Store
	webauthn *webauthn.WebAuthn
	limiter  *limiter
	apiRate  *tokenBucket
	authRate *tokenBucket
}

func New(s *store.Store, cfg Config) (*Service, error) {
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = 7 * 24 * time.Hour
	}
	if cfg.MaxFailedLogins <= 0 {
		cfg.MaxFailedLogins = 3
	}
	if cfg.LockDuration <= 0 {
		cfg.LockDuration = 3 * time.Minute
	}
	if cfg.APIRatePerMinute <= 0 {
		cfg.APIRatePerMinute = 600
	}
	if cfg.AuthRatePerMinute <= 0 {
		cfg.AuthRatePerMinute = 20
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.RPID,
		RPDisplayName: cfg.RPName,
		RPOrigins:     cfg.Origins,
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn config: %w", err)
	}
	return &Service{
		cfg:      cfg,
		store:    s,
		webauthn: wa,
		limiter:  newLimiter(15 * time.Minute),
		// bursts: a page load makes a few requests; a sign-in is 1–2
		apiRate:  newTokenBucket(cfg.APIRatePerMinute, max(cfg.APIRatePerMinute/5, 20)),
		authRate: newTokenBucket(cfg.AuthRatePerMinute, max(cfg.AuthRatePerMinute/2, 5)),
	}, nil
}

// Bootstrap creates the initial admin account when there are no users yet.
// The account must change its password on first sign-in.
func (s *Service) Bootstrap(ctx context.Context) error {
	n, err := s.store.CountUsers(ctx, nil)
	if err != nil || n > 0 {
		return err
	}
	username, err := normalizeUsername(s.cfg.InitialAdminUser)
	if err != nil {
		return fmt.Errorf("initial admin user: %w", err)
	}
	hash, err := hashPassword(s.cfg.InitialAdminPassword)
	if err != nil {
		return err
	}
	u := &model.User{
		Username:           username,
		PasswordHash:       hash,
		Role:               model.RoleAdmin,
		MustChangePassword: true,
		WebAuthnID:         randomBytes(32),
	}
	if err := s.store.CreateUser(ctx, u); err != nil {
		return err
	}
	log.Printf("created initial admin user %q — sign in and change the username/password", username)
	if s.cfg.InitialAdminPassword == "admin" {
		log.Printf("WARNING: its password is the public default \"admin\" — sign in and change it now, " +
			"before the app is reachable by others (or set INITIAL_ADMIN_PASSWORD)")
	}
	return nil
}

func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/config", s.publicConfig)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/auth/me", s.me)
	mux.HandleFunc("POST /api/auth/change-credentials", s.changeCredentials)

	mux.HandleFunc("POST /api/auth/passkey/login/begin", s.passkeyLoginBegin)
	mux.HandleFunc("POST /api/auth/passkey/login/finish", s.passkeyLoginFinish)
	mux.HandleFunc("POST /api/auth/passkeys/register/begin", s.passkeyRegisterBegin)
	mux.HandleFunc("POST /api/auth/passkeys/register/finish", s.passkeyRegisterFinish)
	mux.HandleFunc("PUT /api/auth/passkeys/{id}", s.passkeyRename)
	mux.HandleFunc("DELETE /api/auth/passkeys/{id}", s.passkeyDelete)

	mux.HandleFunc("GET /api/users", s.listUsers)
	mux.HandleFunc("POST /api/users", s.createUser)
	mux.HandleFunc("PUT /api/users/{id}", s.updateUser)
	mux.HandleFunc("POST /api/users/{id}/reset-password", s.resetPassword)
	mux.HandleFunc("DELETE /api/users/{id}", s.deleteUser)
}

// paths reachable without a session
var publicPaths = map[string]bool{
	"/api/health":                    true,
	"/api/auth/config":               true,
	"/api/auth/login":                true,
	"/api/auth/logout":               true,
	"/api/auth/passkey/login/begin":  true,
	"/api/auth/passkey/login/finish": true,
}

// publicRead reports whether r only reads jemaat data, which the public
// (read-only) pages show without signing in (or the Android app's update check). Every write still needs a session.
func publicRead(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	p := r.URL.Path
	return p == "/api/koloms" || p == "/api/app/android" || p == "/api/bipra-categories" || p == "/api/worship-types" || p == "/api/calendar.ics" ||
		p == "/api/services" || strings.HasPrefix(p, "/api/services/") ||
		p == "/api/families" || strings.HasPrefix(p, "/api/families/") ||
		p == "/api/members" || strings.HasPrefix(p, "/api/members/")
}

// adminOnly: user management and the admin pages (Android version, audit log).
func adminOnly(path string) bool {
	return path == "/api/users" || strings.HasPrefix(path, "/api/users/") || strings.HasPrefix(path, "/api/admin/")
}

// managesAllOnly: writes only admins and regular users may make.
func managesAllOnly(path string) bool {
	return path == "/api/koloms" || strings.HasPrefix(path, "/api/koloms/") ||
		path == "/api/worship-types" || strings.HasPrefix(path, "/api/worship-types/")
}

// paths reachable while the user still has to change the initial password
var changePasswordPaths = map[string]bool{
	"/api/auth/me":                 true,
	"/api/auth/logout":             true,
	"/api/auth/change-credentials": true,
}

// Middleware guards /api/*: same-origin check for writes, session check,
// forced password change, and admin-only /api/users and /api/admin/.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasPrefix(path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if isUnsafe(r.Method) && !s.sameOrigin(r) {
			httpx.WriteError(w, http.StatusForbidden, "origin tidak diizinkan")
			return
		}
		if publicPaths[path] {
			next.ServeHTTP(w, r)
			return
		}
		if publicRead(r) {
			// readable by anyone; signed-in users get the full view (see SignedIn)
			u, sess, err := s.authenticate(r)
			if err != nil {
				httpx.ServerError(w, err)
				return
			}
			switch {
			case u != nil && !u.MustChangePassword:
				r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, &principal{user: u, session: sess}))
			case u == nil && hasCookie(r, sessionCookie):
				// expired or revoked session: drop the cookie so the browser
				// stops presenting it; the response is the public view
				// (X-Data-View tells the page, which then signs out)
				s.clearCookie(w, sessionCookie, "/")
			}
			next.ServeHTTP(w, r)
			return
		}
		u, sess, err := s.authenticate(r)
		if err != nil {
			httpx.ServerError(w, err)
			return
		}
		if u == nil {
			httpx.WriteError(w, http.StatusUnauthorized, "silakan masuk terlebih dahulu", "unauthenticated")
			return
		}
		if u.MustChangePassword && !changePasswordPaths[path] {
			httpx.WriteError(w, http.StatusForbidden, "ubah username dan password terlebih dahulu", "password_change_required")
			return
		}
		if adminOnly(path) && !u.IsAdmin() {
			httpx.WriteError(w, http.StatusForbidden, "hanya admin yang dapat mengakses halaman ini", "forbidden")
			return
		}
		// a "kelola data kolom" user changes their kolom's data, not the
		// koloms or jenis ibadah themselves (the data rules are in the API)
		if isUnsafe(r.Method) && !u.ManagesAll() && managesAllOnly(path) {
			httpx.WriteError(w, http.StatusForbidden, u.Scope(), "forbidden")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, &principal{user: u, session: sess})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// publicConfig tells the sign-in page the lockout policy.
func (s *Service) publicConfig(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"maxFailedLogins": s.cfg.MaxFailedLogins,
		"lockSeconds":     int(s.cfg.LockDuration.Seconds()),
	})
}

type ctxKey struct{}

type principal struct {
	user    *model.User
	session *model.Session
}

func hasCookie(r *http.Request, name string) bool {
	c, err := r.Cookie(name)
	return err == nil && c.Value != ""
}

// SignedIn reports whether the request comes from a signed-in user. Handlers
// of public read endpoints use it to decide between the full and the public
// (redacted) view.
func SignedIn(r *http.Request) bool { return current(r) != nil }

// CurrentUser is the signed-in user making the request, or nil.
func CurrentUser(r *http.Request) *model.User {
	if p := current(r); p != nil {
		return p.user
	}
	return nil
}

// WithUser returns r as if u were signed in (for tests of handlers behind
// the middleware).
func WithUser(r *http.Request, u *model.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxKey{}, &principal{user: u}))
}

// ClientIP is the request's client address (behind the configured proxies).
func (s *Service) ClientIP(r *http.Request) string { return s.clientIP(r) }

func current(r *http.Request) *principal {
	p, _ := r.Context().Value(ctxKey{}).(*principal)
	return p
}

// authenticate resolves the session cookie; (nil, nil, nil) means signed out.
func (s *Service) authenticate(r *http.Request) (*model.User, *model.Session, error) {
	c, err := r.Cookie(sessionCookie)
	if errors.Is(err, http.ErrNoCookie) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if c.Value == "" {
		return nil, nil, nil
	}
	sess, err := s.store.GetSession(r.Context(), hashToken(c.Value))
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	u, err := s.store.GetUser(r.Context(), sess.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if u.IsLockedAt(time.Now()) { // sessions are deleted on lock; this covers any race
		return nil, nil, nil
	}
	return u, sess, nil
}

// startSession records the sign-in and gives this browser a session. It
// returns store.ErrLocked, without a session, if the account was locked
// meanwhile.
func (s *Service) startSession(w http.ResponseWriter, r *http.Request, userID bson.ObjectID) (*model.User, error) {
	u, err := s.store.RecordLogin(r.Context(), userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(randomBytes(32))
	now := time.Now().UTC()
	sess := model.Session{
		ID:        hashToken(token),
		UserID:    u.ID,
		UserAgent: r.UserAgent(),
		CreatedAt: now,
		ExpiresAt: now.Add(s.cfg.SessionTTL),
	}
	if err := s.store.CreateSession(r.Context(), sess); err != nil {
		return nil, err
	}
	s.setCookie(w, sessionCookie, token, "/", s.cfg.SessionTTL)
	return u, nil
}

func (s *Service) setCookie(w http.ResponseWriter, name, value, path string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   s.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Service) clearCookie(w http.ResponseWriter, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Path: path, MaxAge: -1, HttpOnly: true, Secure: s.cfg.SecureCookie, SameSite: http.SameSiteLaxMode,
	})
}

func isUnsafe(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

// sameOrigin rejects cross-site writes (CSRF). Browsers always send Origin on
// POST/PUT/DELETE; requests without it (curl, scripts) carry no cookies by accident.
func (s *Service) sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, o := range s.cfg.Origins {
		if strings.EqualFold(o, origin) {
			return true
		}
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}

// clientIP is the address used for rate limiting: the ClientIPHeader value
// when configured; behind a reverse proxy listed in TrustedProxies, the
// right-most X-Forwarded-For entry that is not itself a trusted proxy;
// otherwise the TCP peer address.
func (s *Service) clientIP(r *http.Request) string {
	// a platform edge proxy (Fly.io: Fly-Client-IP) that always overwrites
	// this header is authoritative; only configure it on such a platform
	if s.cfg.ClientIPHeader != "" {
		if v := strings.TrimSpace(r.Header.Get(s.cfg.ClientIPHeader)); v != "" {
			if addr, err := netip.ParseAddr(v); err == nil {
				return addr.Unmap().String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if !s.trusted(host) {
		return host
	}
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(hops[i])
		if hop == "" {
			continue
		}
		if !s.trusted(hop) {
			return hop
		}
		host = hop
	}
	return host
}

func (s *Service) trusted(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	for _, p := range s.cfg.TrustedProxies {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// ParseTrustedProxies parses a comma-separated list of IPs and CIDRs.
func ParseTrustedProxies(list string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, item := range strings.Split(list, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if !strings.Contains(item, "/") {
			addr, err := netip.ParseAddr(item)
			if err != nil {
				return nil, fmt.Errorf("trusted proxy %q: %w", item, err)
			}
			out = append(out, netip.PrefixFrom(addr.Unmap(), addr.Unmap().BitLen()))
			continue
		}
		p, err := netip.ParsePrefix(item)
		if err != nil {
			return nil, fmt.Errorf("trusted proxy %q: %w", item, err)
		}
		out = append(out, p.Masked())
	}
	return out, nil
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand never fails on supported platforms
	}
	return b
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
