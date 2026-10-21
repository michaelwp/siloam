package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/model"
	"siloam/internal/store"
)

// testService needs a MongoDB (MONGODB_TEST_URI, default local) and uses a
// throwaway database; skipped when none is reachable.
func testService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	base := os.Getenv("MONGODB_TEST_URI")
	if base == "" {
		base = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	db := "siloam_test_" + bson.NewObjectID().Hex()
	st, err := store.Open(ctx, strings.TrimRight(base, "/")+"/"+db, "")
	if err != nil {
		t.Skipf("no MongoDB: %v", err)
	}
	t.Cleanup(func() {
		_ = st.DropDatabase(context.Background())
		_ = st.Close(context.Background())
	})
	svc, err := New(st, Config{RPID: "localhost", RPName: "test", Origins: []string{"http://localhost"}})
	if err != nil {
		t.Fatal(err)
	}
	return svc, st
}

func TestPublicReadSessionHandling(t *testing.T) {
	svc, st := testService(t)
	var signedIn bool
	h := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signedIn = SignedIn(r)
		w.WriteHeader(http.StatusNoContent)
	}))
	do := func(cookie string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/api/members", nil)
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}

	// visitor without a cookie: public, no cookie changes
	rec := do("")
	if rec.Code != http.StatusNoContent || signedIn || rec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("visitor: code=%d signedIn=%v set-cookie=%q", rec.Code, signedIn, rec.Header().Get("Set-Cookie"))
	}

	// stale/unknown session cookie: public view, and the cookie is cleared
	rec = do("expired-or-revoked")
	if rec.Code != http.StatusNoContent || signedIn || !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("stale cookie: code=%d signedIn=%v set-cookie=%q", rec.Code, signedIn, rec.Header().Get("Set-Cookie"))
	}

	// valid session: full view
	u := &model.User{Username: "staf", Role: model.RoleUser, WebAuthnID: []byte("staf")}
	if err := st.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	token := "valid-token"
	now := time.Now().UTC()
	if err := st.CreateSession(context.Background(), model.Session{ID: hashToken(token), UserID: u.ID, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	rec = do(token)
	if rec.Code != http.StatusNoContent || !signedIn || rec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("valid session: code=%d signedIn=%v set-cookie=%q", rec.Code, signedIn, rec.Header().Get("Set-Cookie"))
	}
}

func TestAdminPagesNeedTheAdminRole(t *testing.T) {
	svc, st := testService(t)
	h := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	session := func(username, role string) string {
		u := &model.User{Username: username, Role: role, WebAuthnID: []byte(username)}
		if err := st.CreateUser(context.Background(), u); err != nil {
			t.Fatal(err)
		}
		now := time.Now().UTC()
		token := "token-" + username
		if err := st.CreateSession(context.Background(), model.Session{ID: hashToken(token), UserID: u.ID, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
		return token
	}
	staff, admin := session("staf", model.RoleUser), session("admin", model.RoleAdmin)
	for _, path := range []string{"/api/admin/audit", "/api/admin/android"} {
		for token, want := range map[string]int{"": http.StatusUnauthorized, staff: http.StatusForbidden, admin: http.StatusNoContent} {
			r := httptest.NewRequest(http.MethodGet, path, nil)
			if token != "" {
				r.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, r)
			if rec.Code != want {
				t.Errorf("%s with %q: %d, want %d", path, token, rec.Code, want)
			}
		}
	}
}
