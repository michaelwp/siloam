// Package audit records user activities: every change made through the API
// (POST/PUT/DELETE under /api/) and every sign-in and sign-out, with who did
// it, what it was about, whether it worked, the client IP and the time.
// Request bodies are never stored, so passwords cannot end up in the log.
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/model"
)

// Category is a group of activities, for the log page's filter.
type Category struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Categories in the order the log page lists them.
var Categories = []Category{
	{"masuk", "Masuk & keluar"},
	{"akun", "Akun & passkey"},
	{"pengguna", "Pengguna"},
	{"kolom", "Kolom"},
	{"keluarga", "Keluarga"},
	{"anggota", "Anggota"},
	{"jenis-ibadah", "Jenis ibadah"},
	{"jadwal", "Jadwal ibadah"},
	{"android", "Aplikasi Android"},
	{"bipra", "Kategori BIPRA"},
}

type route struct{ category, action string }

// routes are the recorded requests, by their ServeMux pattern. Others (e.g.
// the first half of a passkey ceremony, unknown paths) are not recorded.
var routes = map[string]route{
	"POST /api/auth/login":                    {"masuk", "Masuk dengan password"},
	"POST /api/auth/passkey/login/finish":     {"masuk", "Masuk dengan passkey"},
	"POST /api/auth/logout":                   {"masuk", "Keluar"},
	"POST /api/auth/change-credentials":       {"akun", "Ubah username/password"},
	"POST /api/auth/passkeys/register/finish": {"akun", "Tambah passkey"},
	"PUT /api/auth/passkeys/{id}":             {"akun", "Ganti nama passkey"},
	"DELETE /api/auth/passkeys/{id}":          {"akun", "Hapus passkey"},
	"POST /api/users":                         {"pengguna", "Tambah pengguna"},
	"PUT /api/users/{id}":                     {"pengguna", "Ubah pengguna"},
	"POST /api/users/{id}/reset-password":     {"pengguna", "Reset password pengguna"},
	"DELETE /api/users/{id}":                  {"pengguna", "Hapus pengguna"},
	"POST /api/koloms":                        {"kolom", "Tambah kolom"},
	"DELETE /api/koloms/{id}":                 {"kolom", "Hapus kolom"},
	"POST /api/families":                      {"keluarga", "Tambah keluarga"},
	"PUT /api/families/{id}":                  {"keluarga", "Ubah keluarga"},
	"DELETE /api/families/{id}":               {"keluarga", "Hapus keluarga"},
	"POST /api/members":                       {"anggota", "Tambah anggota"},
	"PUT /api/members/{id}":                   {"anggota", "Ubah anggota"},
	"DELETE /api/members/{id}":                {"anggota", "Hapus anggota"},
	"POST /api/worship-types":                 {"jenis-ibadah", "Tambah jenis ibadah"},
	"PUT /api/worship-types/{id}":             {"jenis-ibadah", "Ubah jenis ibadah"},
	"DELETE /api/worship-types/{id}":          {"jenis-ibadah", "Hapus jenis ibadah"},
	"POST /api/services":                      {"jadwal", "Tambah jadwal ibadah"},
	"PUT /api/services/{id}":                  {"jadwal", "Ubah jadwal ibadah"},
	"DELETE /api/services/{id}":               {"jadwal", "Hapus jadwal ibadah"},
	"PUT /api/admin/android":                  {"android", "Ubah versi aplikasi Android"},
	"DELETE /api/admin/android":               {"android", "Kembalikan versi Android ke pengaturan server"},
	"POST /api/admin/bipra":                   {"bipra", "Tambah kategori BIPRA"},
	"PUT /api/admin/bipra/{id}":               {"bipra", "Ubah kategori BIPRA"},
	"DELETE /api/admin/bipra/{id}":            {"bipra", "Hapus kategori BIPRA"},
	"PUT /api/admin/bipra/order":              {"bipra", "Ubah urutan kategori BIPRA"},
}

// Store is where entries go (the MongoDB store; a fake in tests).
type Store interface {
	InsertAudit(ctx context.Context, e *model.AuditEntry) error
	// AuditLabel names a document before it is deleted ("" if unknown).
	AuditLabel(ctx context.Context, kind string, id bson.ObjectID) string
}

// Recorder is the audit middleware.
type Recorder struct {
	Store    Store
	ClientIP func(*http.Request) string
	User     func(*http.Request) *model.User // the signed-in user, or nil
}

type noteKey struct{}

// note lets a handler say who acted when the request has no session yet
// (sign-in) or no longer has one (sign-out).
type note struct {
	user    *model.User
	attempt string
}

func noteOf(r *http.Request) *note {
	n, _ := r.Context().Value(noteKey{}).(*note)
	return n
}

// SetUser records u as the one who acted (a sign-in that worked, a sign-out).
func SetUser(r *http.Request, u *model.User) {
	if n := noteOf(r); n != nil {
		n.user = u
	}
}

// SetAttempt records the username a failed sign-in tried.
func SetAttempt(r *http.Request, username string) {
	if n := noteOf(r); n != nil {
		n.attempt = username
	}
}

// statusWriter keeps the status and (for names) the start of a JSON body.
type statusWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

const maxCapture = 64 << 10

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if room := maxCapture - w.body.Len(); room > 0 {
		w.body.Write(b[:min(len(b), room)])
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func isWrite(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

// Middleware records the writes under /api/ that next (the router) handles.
// It goes inside the auth middleware, which has already refused requests
// without the needed session or role.
func (rec *Recorder) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || !isWrite(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		n := &note{}
		r = r.WithContext(context.WithValue(r.Context(), noteKey{}, n))
		// a delete takes the name with it: look it up first
		kind, id := pathTarget(r.URL.Path)
		label := ""
		if r.Method == http.MethodDelete && !id.IsZero() {
			label = rec.Store.AuditLabel(r.Context(), kind, id)
		}
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)

		rt, ok := routes[r.Pattern] // set by the ServeMux on this request
		if !ok {
			return
		}
		e := &model.AuditEntry{
			At:       time.Now().UTC(),
			Category: rt.category,
			Action:   rt.action,
			Method:   r.Method,
			Path:     r.URL.Path,
			Status:   sw.status,
			IP:       rec.ClientIP(r),
		}
		if e.Status == 0 {
			e.Status = http.StatusOK
		}
		u := n.user
		if u == nil && rec.User != nil {
			u = rec.User(r)
		}
		switch {
		case u != nil:
			e.UserID, e.Username = &u.ID, u.Username
		case n.attempt != "":
			e.Username = n.attempt
		}
		if !id.IsZero() {
			e.TargetID = id.Hex()
		}
		e.Target = label
		if e.Target == "" && e.Status < 300 && r.Method != http.MethodDelete {
			e.Target = labelFromBody(kind, sw.body.Bytes())
		}
		if e.Target == "" && kind == "auth" && strings.HasSuffix(r.URL.Path, "/register/finish") {
			e.Target = r.URL.Query().Get("name") // the passkey's name
		}
		// the response has been written; the entry must still be stored
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
		defer cancel()
		if err := rec.Store.InsertAudit(ctx, e); err != nil {
			log.Printf("audit: %s %s: %v", e.Method, e.Path, err)
		}
	})
}

// pathTarget splits /api/<kind>/<id>… (or /api/admin/<kind>/<id>) into the
// kind and (if it is one) the id.
func pathTarget(path string) (string, bson.ObjectID) {
	parts := strings.Split(strings.TrimPrefix(path, "/api/"), "/")
	if parts[0] == "admin" && len(parts) > 1 {
		parts = parts[1:]
	}
	if len(parts) < 2 {
		return parts[0], bson.ObjectID{}
	}
	id, _ := bson.ObjectIDFromHex(parts[1]) // zero when it is not an id (e.g. /api/auth/login)
	return parts[0], id
}

// labelFromBody names what a create/update returned.
func labelFromBody(kind string, body []byte) string {
	var v struct {
		Number        int    `json:"number"`
		Name          string `json:"name"`
		Username      string `json:"username"`
		Host          string `json:"host"`
		Date          string `json:"date"`
		LatestVersion string `json:"latestVersion"`
	}
	if json.Unmarshal(body, &v) != nil {
		return ""
	}
	switch kind {
	case "koloms":
		if v.Number > 0 {
			return fmt.Sprintf("Kolom %d", v.Number)
		}
	case "families":
		if v.Name != "" {
			return "Kel. " + v.Name
		}
	case "services":
		label := v.Date
		if len(label) >= 10 {
			label = label[:10]
		}
		if v.Host != "" {
			label = "Kel. " + v.Host + " · " + label
		}
		return label
	case "users":
		return v.Username
	case "android":
		if v.LatestVersion != "" {
			return "Versi " + v.LatestVersion
		}
		return ""
	}
	return v.Name
}
