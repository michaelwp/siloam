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
		r := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
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

	// jemaat data needs a session
	r := httptest.NewRequest(http.MethodGet, "/api/members", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("visitor reads members: %d", rec.Code)
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

func TestKolomUserCannotChangeKolomsOrJenisIbadah(t *testing.T) {
	svc, st := testService(t)
	h := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	u := &model.User{Username: "kolom3", Role: model.RoleScoped, Koloms: []int{3}, WebAuthnID: []byte("kolom3")}
	if err := st.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := st.CreateSession(context.Background(), model.Session{ID: hashToken("k3"), UserID: u.ID, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/api/koloms", http.StatusForbidden},
		{http.MethodDelete, "/api/koloms/abc", http.StatusForbidden},
		{http.MethodPost, "/api/worship-types", http.StatusForbidden},
		{http.MethodPut, "/api/worship-types/abc", http.StatusForbidden},
		{http.MethodGet, "/api/koloms", http.StatusNoContent},
		{http.MethodPost, "/api/families", http.StatusNoContent}, // the API checks the kolom
		{http.MethodGet, "/api/users", http.StatusForbidden},
	} {
		r := httptest.NewRequest(c.method, c.path, nil)
		r.Header.Set("Origin", "http://example.com")
		r.Host = "example.com"
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "k3"})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != c.want {
			t.Errorf("%s %s: %d, want %d", c.method, c.path, rec.Code, c.want)
		}
	}
}

func TestViewerChangesOnlyTheirAccount(t *testing.T) {
	svc, st := testService(t)
	h := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	u := &model.User{Username: "lihat", Role: model.RoleViewer, WebAuthnID: []byte("lihat")}
	if err := st.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := st.CreateSession(context.Background(), model.Session{ID: hashToken("v"), UserID: u.ID, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/api/families", http.StatusNoContent},
		{http.MethodPost, "/api/families", http.StatusForbidden},
		{http.MethodPut, "/api/members/abc", http.StatusForbidden},
		{http.MethodDelete, "/api/services/abc", http.StatusForbidden},
		{http.MethodPost, "/api/koloms", http.StatusForbidden},
		{http.MethodPut, "/api/auth/passkeys/abc", http.StatusNoContent},
		{http.MethodPost, "/api/auth/change-credentials", http.StatusNoContent},
	} {
		r := httptest.NewRequest(c.method, c.path, nil)
		r.Header.Set("Origin", "http://example.com")
		r.Host = "example.com"
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "v"})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != c.want {
			t.Errorf("%s %s: %d, want %d", c.method, c.path, rec.Code, c.want)
		}
	}
}

// A user manages (or sees) koloms and/or BIPRA categories.
func TestRoleScope(t *testing.T) {
	svc, st := testService(t)
	ctx := context.Background()
	if err := st.SeedBipra(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateKolomIfMissing(ctx, 3); err != nil {
		t.Fatal(err)
	}
	rules, _ := st.BipraRules(ctx)
	cat := rules[0].ID.Hex()
	r := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	for _, c := range []struct {
		what   string
		role   string
		koloms []int
		bipras []string
		ok     bool
	}{
		{"kolom user", model.RoleScoped, []int{3}, nil, true},
		{"BIPRA user", model.RoleScoped, nil, []string{cat}, true},
		{"both", model.RoleScoped, []int{3}, []string{cat}, true},
		{"neither", model.RoleScoped, nil, nil, false},
		{"viewer of both", model.RoleViewer, []int{3}, []string{cat}, true},
		{"a kolom that does not exist", model.RoleScoped, []int{99}, nil, false},
		{"viewer of everything", model.RoleViewer, nil, nil, true},
	} {
		if _, _, err := svc.roleScope(r, c.role, c.koloms, c.bipras); (err == nil) != c.ok {
			t.Errorf("%s: %v", c.what, err)
		}
	}
}

// An admin signs a user out everywhere, giving the reason.
func TestAdminLogsOutUser(t *testing.T) {
	svc, st := testService(t)
	ctx := context.Background()
	mux := http.NewServeMux()
	svc.Register(mux)
	h := svc.Middleware(mux)
	admin := &model.User{Username: "admin", Role: model.RoleAdmin, WebAuthnID: []byte("admin")}
	budi := &model.User{Username: "budi", Role: model.RoleUser, WebAuthnID: []byte("budi")}
	for _, u := range []*model.User{admin, budi} {
		if err := st.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	for id, u := range map[string]*model.User{"a": admin, "b1": budi, "b2": budi} {
		if err := st.CreateSession(ctx, model.Session{ID: hashToken(id), UserID: u.ID, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	post := func(id bson.ObjectID, body string) int {
		r := httptest.NewRequest(http.MethodPost, "/api/users/"+id.Hex()+"/logout", strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a"})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec.Code
	}
	if c := post(budi.ID, `{"reason":""}`); c != http.StatusBadRequest {
		t.Errorf("without a reason: %d", c)
	}
	if c := post(admin.ID, `{"reason":"tes saja"}`); c != http.StatusBadRequest {
		t.Errorf("themselves: %d", c)
	}
	if c := post(budi.ID, `{"reason":"HP hilang"}`); c != http.StatusOK {
		t.Fatalf("with a reason: %d", c)
	}
	sessions, _ := st.ListActiveSessions(ctx, now)
	if len(sessions) != 1 || sessions[0].UserID != admin.ID {
		t.Errorf("sessions left: %+v", sessions)
	}
}
