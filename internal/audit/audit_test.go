package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/httpx"
	"siloam/internal/model"
)

type fakeStore struct {
	entries []model.AuditEntry
	labels  map[string]string // kind/id → label
}

func (f *fakeStore) InsertAudit(_ context.Context, e *model.AuditEntry) error {
	f.entries = append(f.entries, *e)
	return nil
}

func (f *fakeStore) AuditLabel(_ context.Context, kind string, id bson.ObjectID) string {
	return f.labels[kind+"/"+id.Hex()]
}

func setup(user *model.User) (*fakeStore, http.Handler) {
	fs := &fakeStore{labels: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/members", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": "x", "name": "Albert Supit"})
	})
	mux.HandleFunc("DELETE /api/families/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("PUT /api/services/{id}", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteError(w, http.StatusBadRequest, "tanggal wajib diisi")
	})
	mux.HandleFunc("GET /api/members", func(w http.ResponseWriter, _ *http.Request) { httpx.WriteJSON(w, http.StatusOK, []string{}) })
	mux.HandleFunc("POST /api/auth/passkeys/register/begin", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("POST /api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "ok") {
			SetUser(r, &model.User{ID: bson.NewObjectID(), Username: "budi"})
			w.WriteHeader(http.StatusOK)
			return
		}
		SetAttempt(r, "budi")
		httpx.WriteError(w, http.StatusUnauthorized, "salah")
	})
	rec := &Recorder{
		Store:    fs,
		ClientIP: func(*http.Request) string { return "203.0.113.9" },
		User:     func(*http.Request) *model.User { return user },
	}
	return fs, rec.Middleware(mux)
}

func do(h http.Handler, method, target, body string) {
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, target, strings.NewReader(body)))
}

func TestRecordsChangesWithWhoWhatAndResult(t *testing.T) {
	admin := &model.User{ID: bson.NewObjectID(), Username: "admin"}
	fs, h := setup(admin)
	famID := bson.NewObjectID()
	fs.labels["families/"+famID.Hex()] = "Kel. Umboh"
	svcID := bson.NewObjectID()

	do(h, http.MethodPost, "/api/members", `{"name":"Albert Supit","password":"never-logged"}`)
	do(h, http.MethodDelete, "/api/families/"+famID.Hex(), "")
	do(h, http.MethodPut, "/api/services/"+svcID.Hex(), `{}`)
	do(h, http.MethodGet, "/api/members", "")                       // reads are not recorded
	do(h, http.MethodPost, "/api/auth/passkeys/register/begin", "") // nor ceremony starts
	do(h, http.MethodPost, "/api/unknown", "")                      // nor unknown paths

	if len(fs.entries) != 3 {
		t.Fatalf("entries = %+v", fs.entries)
	}
	add, del, put := fs.entries[0], fs.entries[1], fs.entries[2]
	if add.Action != "Tambah anggota" || add.Category != "anggota" || add.Target != "Albert Supit" || add.Status != 201 ||
		add.Username != "admin" || *add.UserID != admin.ID || add.IP != "203.0.113.9" {
		t.Errorf("create: %+v", add)
	}
	if del.Action != "Hapus keluarga" || del.Target != "Kel. Umboh" || del.TargetID != famID.Hex() || del.Status != 204 {
		t.Errorf("delete (named before it went): %+v", del)
	}
	if put.Action != "Ubah jadwal ibadah" || put.Status != 400 || put.Target != "" {
		t.Errorf("refused update: %+v", put)
	}
}

func TestSignInsNameTheUser(t *testing.T) {
	fs, h := setup(nil) // no session during sign-in
	do(h, http.MethodPost, "/api/auth/login?ok", `{"username":"budi","password":"secret"}`)
	do(h, http.MethodPost, "/api/auth/login", `{"username":"budi","password":"wrong"}`)
	if len(fs.entries) != 2 {
		t.Fatalf("entries = %+v", fs.entries)
	}
	ok, failed := fs.entries[0], fs.entries[1]
	if ok.Username != "budi" || ok.UserID == nil || ok.Status != 200 || ok.Action != "Masuk dengan password" {
		t.Errorf("sign-in: %+v", ok)
	}
	if failed.Username != "budi" || failed.UserID != nil || failed.Status != 401 {
		t.Errorf("failed sign-in: %+v", failed)
	}
}
