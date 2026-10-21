package api

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/audit"
	"siloam/internal/auth"
	"siloam/internal/model"
	"siloam/internal/store"
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
	f := store.AuditFilter{Username: q.Get("username"), Category: q.Get("category"), Limit: 50}
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
	entries, err := a.store.ListAudit(r.Context(), f)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// auditMeta gives the log page its filter choices.
func (a *API) auditMeta(w http.ResponseWriter, r *http.Request) {
	names, err := a.store.AuditUsernames(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": audit.Categories, "usernames": names})
}
