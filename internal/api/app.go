package api

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/audit"
	"siloam/internal/auth"
	"siloam/internal/model"
)

// ---- Android app version ----

// release is the one in effect: set on the admin page, else the server's
// ANDROID_* settings ([API.AndroidDefault]); the second result says which
// ("database", "env" or "none").
func (a *API) release(r *http.Request) (model.AndroidRelease, string, error) {
	saved, err := a.store.GetAndroidRelease(r.Context())
	switch {
	case err != nil:
		return model.AndroidRelease{}, "", err
	case saved != nil:
		return *saved, "database", nil
	case a.AndroidDefault.LatestVersion != "":
		return a.AndroidDefault, "env", nil
	default:
		return model.AndroidRelease{}, "none", nil
	}
}

// androidRelease answers the app's update check (public): only the version
// and the link.
func (a *API) androidRelease(w http.ResponseWriter, r *http.Request) {
	rel, _, err := a.release(r)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, model.AndroidRelease{LatestVersion: rel.LatestVersion, DownloadURL: rel.DownloadURL})
}

type releaseView struct {
	model.AndroidRelease
	Source string               `json:"source"` // database | env | none
	Server model.AndroidRelease `json:"server"` // the ANDROID_* settings
}

// adminAndroid shows the admin page what is in effect and where it comes from.
func (a *API) adminAndroid(w http.ResponseWriter, r *http.Request) {
	rel, source, err := a.release(r)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, releaseView{rel, source, a.AndroidDefault})
}

// setAndroid saves the release from the admin page; both fields empty turns
// the update notice off (whatever ANDROID_* says).
func (a *API) setAndroid(w http.ResponseWriter, r *http.Request) {
	var in struct {
		LatestVersion string `json:"latestVersion"`
		DownloadURL   string `json:"downloadUrl"`
	}
	if !decode(w, r, &in) {
		return
	}
	now := time.Now().UTC()
	rel := model.AndroidRelease{LatestVersion: in.LatestVersion, DownloadURL: in.DownloadURL, UpdatedAt: &now}
	if u := auth.CurrentUser(r); u != nil {
		rel.UpdatedBy = u.Username
	}
	if err := rel.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.store.SetAndroidRelease(r.Context(), rel); err != nil {
		serverError(w, err)
		return
	}
	a.adminAndroid(w, r)
}

// clearAndroid drops the admin page's setting: ANDROID_* applies again.
func (a *API) clearAndroid(w http.ResponseWriter, r *http.Request) {
	if err := a.store.ClearAndroidRelease(r.Context()); err != nil {
		serverError(w, err)
		return
	}
	a.adminAndroid(w, r)
}

// ---- audit log ----

// listAudit pages through the log, newest first:
// ?username=&category=&result=ok|failed&from=YYYY-MM-DD&to=YYYY-MM-DD&before=<id>&limit=
// (dates are in APP_TIMEZONE; "to" includes that day).
func (a *API) listAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := audit.Filter{Username: q.Get("username"), Category: q.Get("category"), Limit: 50}
	switch q.Get("result") {
	case "ok":
		failed := false
		f.Failed = &failed
	case "failed":
		failed := true
		f.Failed = &failed
	}
	day := func(key string) (time.Time, bool) {
		v := q.Get(key)
		if v == "" {
			return time.Time{}, true
		}
		t, err := time.ParseInLocation("2006-01-02", v, a.loc)
		if err != nil {
			writeError(w, http.StatusBadRequest, "tanggal tidak valid: "+v)
			return time.Time{}, false
		}
		return t, true
	}
	var ok bool
	if f.From, ok = day("from"); !ok {
		return
	}
	if f.To, ok = day("to"); !ok {
		return
	}
	if !f.To.IsZero() {
		f.To = f.To.AddDate(0, 0, 1)
	}
	if v := q.Get("before"); v != "" {
		id, err := bson.ObjectIDFromHex(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "before tidak valid")
			return
		}
		f.Before = id
	}
	if a.AuditLog == nil {
		writeJSON(w, http.StatusOK, []model.AuditEntry{})
		return
	}
	entries, err := a.AuditLog.List(f)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// auditMeta gives the log page its filter choices.
func (a *API) auditMeta(w http.ResponseWriter, _ *http.Request) {
	names := []string{}
	if a.AuditLog != nil {
		var err error
		if names, err = a.AuditLog.Usernames(); err != nil {
			serverError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": audit.Categories, "usernames": names})
}

// LoginStats is the admins' user statistics for one day (APP_TIMEZONE).
type LoginStats struct {
	Date    string        `json:"date"`    // YYYY-MM-DD
	Users   int           `json:"users"`   // distinct users who signed in that day
	Web     int           `json:"web"`     // sign-ins from the web app
	Android int           `json:"android"` // sign-ins from the Android app
	Logins  []model.Login `json:"logins"`  // newest first
	// Active: who is signed in now (unexpired sessions), per user and source
	Active []ActiveUser `json:"active"`
}

type ActiveUser struct {
	UserID   bson.ObjectID   `json:"userId"`
	Username string          `json:"username"`
	Role     string          `json:"role"`
	Koloms   []int           `json:"koloms,omitempty"`
	BipraIDs []bson.ObjectID `json:"bipraIds,omitempty"`
	Web      int             `json:"web"`     // signed-in browsers
	Android  int             `json:"android"` // signed-in phones
	Since    time.Time       `json:"since"`   // the newest of those sign-ins
}

// loginStats: GET /api/admin/logins?date=YYYY-MM-DD (default today).
func (a *API) loginStats(w http.ResponseWriter, r *http.Request) {
	now := a.now().In(a.loc)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, a.loc)
	if v := r.URL.Query().Get("date"); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, a.loc)
		if err != nil {
			writeError(w, http.StatusBadRequest, "tanggal tidak valid: "+v)
			return
		}
		day = t
	}
	logins, err := a.store.ListLogins(r.Context(), day.UTC(), day.AddDate(0, 0, 1).UTC())
	if err != nil {
		serverError(w, err)
		return
	}
	out := LoginStats{Date: day.Format("2006-01-02"), Logins: logins, Active: []ActiveUser{}}
	seen := map[bson.ObjectID]bool{}
	for _, l := range logins {
		seen[l.UserID] = true
		if l.Source == model.LoginAndroid {
			out.Android++
		} else {
			out.Web++
		}
	}
	out.Users = len(seen)

	sessions, err := a.store.ListActiveSessions(r.Context(), time.Now().UTC())
	if err != nil {
		serverError(w, err)
		return
	}
	users, err := a.store.ListUsers(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	byID := map[bson.ObjectID]model.User{}
	for _, u := range users {
		byID[u.ID] = u
	}
	idx := map[bson.ObjectID]int{}
	for _, s := range sessions { // newest first, so Since is the first one's
		u, ok := byID[s.UserID]
		if !ok {
			continue
		}
		i, ok := idx[u.ID]
		if !ok {
			i = len(out.Active)
			idx[u.ID] = i
			out.Active = append(out.Active, ActiveUser{UserID: u.ID, Username: u.Username, Role: u.Role, Koloms: u.Koloms, BipraIDs: u.BipraIDs, Since: s.CreatedAt})
		}
		if s.Source == model.LoginAndroid {
			out.Active[i].Android++
		} else {
			out.Active[i].Web++
		}
	}
	writeJSON(w, http.StatusOK, out)
}
