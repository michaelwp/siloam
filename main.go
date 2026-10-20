// Command siloam serves the Jemaat Kolom 9 database app: a REST API backed by
// MongoDB plus the React frontend embedded in the binary.
//
//	siloam                                  # serve on :8080
//	siloam -import data.xlsx [-replace]     # load the spreadsheet, then exit
//	siloam -copy-from <mongodb-uri> [-replace]  # copy all data into MONGODB_URI, then exit
//
// Settings come from environment variables, falling back to a .env file in the
// working directory (ENV_FILE to use another path); see .env.example.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"siloam/internal/api"
	"siloam/internal/auth"
	"siloam/internal/config"
	"siloam/internal/importer"
	"siloam/internal/store"
	"siloam/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	envFile := env("ENV_FILE", ".env")
	loaded, err := config.LoadDotEnv(envFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", envFile, err)
	}

	addr := flag.String("addr", env("ADDR", ":8080"), "HTTP listen address")
	mongoURI := flag.String("mongo", env("MONGODB_URI", "mongodb://localhost:27017/"+store.DefaultDatabase), "MongoDB connection URI (MONGODB_URI)")
	dbName := flag.String("db", "", "MongoDB database name (default: from the URI path, else \""+store.DefaultDatabase+"\")")
	importPath := flag.String("import", "", "import an .xlsx workbook and exit")
	copyFrom := flag.String("copy-from", "", "copy families, members and users from this MongoDB URI into MONGODB_URI and exit")
	resetUser := flag.String("reset-user", "", "unlock this account, set a temporary password, print it and exit")
	copyOnly := flag.String("collections", strings.Join(store.CopyCollections, ","), "with -copy-from: comma-separated collections to copy")
	replace := flag.Bool("replace", false, "with -import / -copy-from: delete existing data first")
	flag.Parse()

	if loaded {
		log.Printf("loaded settings from %s", envFile)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// generous timeout: in containers MongoDB may still be starting
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	s, err := store.Open(connectCtx, *mongoURI, *dbName)
	cancel()
	if err != nil {
		return fmt.Errorf("connect mongodb %s: %w", redact(*mongoURI), err)
	}
	defer func() {
		if err := s.Close(context.Background()); err != nil {
			log.Printf("disconnect mongodb: %v", err)
		}
	}()
	log.Printf("mongodb successfully connected")

	if *copyFrom != "" {
		return runCopy(ctx, s, *copyFrom, strings.Split(*copyOnly, ","), *replace)
	}
	if *importPath != "" {
		return runImport(ctx, s, *importPath, *replace)
	}

	authSvc, err := auth.New(s, authConfig(*addr))
	if err != nil {
		return err
	}
	if *resetUser != "" {
		password, err := authSvc.ResetUser(ctx, *resetUser)
		if err != nil {
			return err
		}
		fmt.Printf("\nAccount %q is unlocked.\nTemporary password: %s\nIt must be changed at the next sign-in.\n\n", *resetUser, password)
		return nil
	}
	if err := authSvc.Bootstrap(ctx); err != nil {
		return fmt.Errorf("create initial admin: %w", err)
	}

	mux := http.NewServeMux()
	authSvc.Register(mux)
	api.New(s).Register(mux)
	mux.Handle("/", spaHandler(web.Dist()))

	handler := logRequests(securityHeaders(authSvc.RateLimit(authSvc.Middleware(mux)), env("COOKIE_SECURE", "false") == "true"))
	srv := &http.Server{Addr: *addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	log.Printf("listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func runImport(ctx context.Context, s *store.Store, file string, replace bool) error {
	empty, err := s.IsEmpty(ctx)
	if err != nil {
		return err
	}
	if !empty {
		if !replace {
			return errors.New("database already has data; pass -replace to overwrite it")
		}
		if err := s.Reset(ctx); err != nil {
			return err
		}
	}
	res, err := importer.Import(ctx, s, file)
	if err != nil {
		return fmt.Errorf("import: %w", err)
	}
	log.Printf("imported %d families, %d members", res.Families, res.Members)
	return nil
}

func runCopy(ctx context.Context, dst *store.Store, srcURI string, names []string, replace bool) error {
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	src, err := store.Connect(connectCtx, srcURI, "")
	cancel()
	if err != nil {
		return fmt.Errorf("connect source %s: %w", redact(srcURI), err)
	}
	defer func() { _ = src.Close(context.Background()) }()

	log.Printf("copying from %s (database %q)", redact(srcURI), src.Database())
	res, err := dst.CopyFrom(ctx, src, names, replace)
	for _, name := range store.CopyCollections {
		if n, ok := res[name]; ok {
			log.Printf("  %-9s %d documents", name, n)
		}
	}
	if err != nil {
		return err
	}
	log.Printf("copy complete (sessions are never copied)")
	return nil
}

// redact hides the password in a MongoDB URI for logging.
func redact(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return "(invalid uri)"
	}
	u.RawQuery = ""
	return u.Redacted()
}

// spaHandler serves the built frontend. Hashed files under assets/ are cached
// for a year; a missing asset is a 404 (never index.html, which would get
// cached as that asset). Any other unknown path — a client-side route like
// /keluarga — gets index.html, uncached so a redeploy is picked up. Directory
// listings are never served.
func spaHandler(dist fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if info, err := fs.Stat(dist, p); err == nil && info.Mode().IsRegular() && p != "index.html" {
			if strings.HasPrefix(p, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			serveFile(w, r, dist, p)
			return
		}
		if p == "assets" || strings.HasPrefix(p, "assets/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		serveFile(w, r, dist, "index.html")
	})
}

// serveFile writes one file from dist (ServeFileFS would redirect
// "/index.html" requests and can list directories; this does neither).
func serveFile(w http.ResponseWriter, r *http.Request, dist fs.FS, name string) {
	f, err := dist.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, name, info.ModTime(), rs)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}

// authConfig reads sign-in settings from the environment:
//
//	WEBAUTHN_RP_ID          passkey domain (default "localhost")
//	WEBAUTHN_ORIGINS        comma-separated origins the browser uses, e.g. https://jemaat.example.org
//	                        (default http://localhost:<port>, http://localhost:5173)
//	COOKIE_SECURE           "true" behind HTTPS
//	INITIAL_ADMIN_USER      first admin account, created when there are no users (default "admin")
//	INITIAL_ADMIN_PASSWORD  its temporary password, must be changed on first sign-in (default "admin")
//	MAX_FAILED_LOGINS       wrong passwords in a row before an account locks (default 3)
//	LOCK_DURATION           how long the lock lasts, e.g. 3m, 90s, 1h (default 3m)
//	API_RATE_LIMIT          /api requests per client IP per minute (default 600)
//	AUTH_RATE_LIMIT         sign-in attempts per client IP per minute (default 20)
//	TRUSTED_PROXIES         reverse proxies (IPs/CIDRs) whose X-Forwarded-For gives the client IP
//	CLIENT_IP_HEADER        header with the client IP set by the platform's edge (Fly.io: Fly-Client-IP)
func authConfig(addr string) auth.Config {
	_, port, _ := net.SplitHostPort(addr)
	if port == "" {
		port = "8080"
	}
	origins := []string{"http://localhost:" + port, "http://localhost:5173"}
	if v := os.Getenv("WEBAUTHN_ORIGINS"); v != "" {
		origins = nil
		for _, o := range strings.Split(v, ",") {
			if o = strings.TrimRight(strings.TrimSpace(o), "/"); o != "" {
				origins = append(origins, o)
			}
		}
	}
	proxies, err := auth.ParseTrustedProxies(os.Getenv("TRUSTED_PROXIES"))
	if err != nil {
		log.Fatalf("TRUSTED_PROXIES: %v", err)
	}
	lockFor, err := time.ParseDuration(env("LOCK_DURATION", "3m"))
	if err != nil || lockFor < time.Second {
		log.Printf("invalid LOCK_DURATION (use e.g. 3m, 90s, 1h), using 3m")
		lockFor = 3 * time.Minute
	}
	maxFailed, err := strconv.Atoi(env("MAX_FAILED_LOGINS", "3"))
	if err != nil || maxFailed < 1 {
		log.Printf("invalid MAX_FAILED_LOGINS, using 3")
		maxFailed = 3
	}
	return auth.Config{
		RPID:                 env("WEBAUTHN_RP_ID", "localhost"),
		RPName:               "Jemaat Kolom 9",
		Origins:              origins,
		SecureCookie:         env("COOKIE_SECURE", "false") == "true",
		InitialAdminUser:     env("INITIAL_ADMIN_USER", "admin"),
		InitialAdminPassword: env("INITIAL_ADMIN_PASSWORD", "admin"),
		MaxFailedLogins:      maxFailed,
		LockDuration:         lockFor,
		APIRatePerMinute:     envInt("API_RATE_LIMIT", 600),
		AuthRatePerMinute:    envInt("AUTH_RATE_LIMIT", 20),
		TrustedProxies:       proxies,
		ClientIPHeader:       os.Getenv("CLIENT_IP_HEADER"),
	}
}

// securityHeaders sets defensive headers; hsts (when serving over HTTPS) tells
// browsers to never use plain HTTP for this site.
func securityHeaders(next http.Handler, hsts bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		if hsts {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		// member data: keep pages and API responses out of search engines
		h.Set("X-Robots-Tag", "noindex, nofollow")
		next.ServeHTTP(w, r)
	})
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		log.Printf("invalid %s=%q, using %d", key, v, def)
		return def
	}
	return n
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
