// Package api exposes the REST endpoints under /api.
package api

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/auth"
	"siloam/internal/httpx"
	"siloam/internal/model"
	"siloam/internal/store"
)

type API struct {
	store *store.Store
	loc   *time.Location // local time of the services (WITA)
	// AndroidDefault is the newest Android app per the server's ANDROID_*
	// settings; the admin page's setting, once saved, takes precedence
	AndroidDefault model.AndroidRelease
}

func New(s *store.Store, loc *time.Location) *API { return &API{store: s, loc: loc} }

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", a.health)
	mux.HandleFunc("GET /api/app/android", a.androidRelease)

	// admin only (the auth middleware guards /api/admin/)
	mux.HandleFunc("GET /api/admin/android", a.adminAndroid)
	mux.HandleFunc("PUT /api/admin/android", a.setAndroid)
	mux.HandleFunc("DELETE /api/admin/android", a.clearAndroid)
	mux.HandleFunc("GET /api/admin/audit", a.listAudit)
	mux.HandleFunc("GET /api/admin/audit/meta", a.auditMeta)
	mux.HandleFunc("POST /api/admin/bipra", a.createBipra)
	mux.HandleFunc("PUT /api/admin/bipra/order", a.reorderBipra)
	mux.HandleFunc("PUT /api/admin/bipra/{id}", a.updateBipra)
	mux.HandleFunc("DELETE /api/admin/bipra/{id}", a.deleteBipra)
	mux.HandleFunc("GET /api/bipra-categories", a.listBipra)

	mux.HandleFunc("GET /api/koloms", a.listKoloms)
	mux.HandleFunc("POST /api/koloms", a.createKolom)
	mux.HandleFunc("DELETE /api/koloms/{id}", a.deleteKolom)

	mux.HandleFunc("GET /api/families", a.listFamilies)
	mux.HandleFunc("POST /api/families", a.createFamily)
	mux.HandleFunc("GET /api/families/{id}", a.getFamily)
	mux.HandleFunc("PUT /api/families/{id}", a.updateFamily)
	mux.HandleFunc("DELETE /api/families/{id}", a.deleteFamily)

	mux.HandleFunc("GET /api/members", a.listMembers)
	mux.HandleFunc("POST /api/members", a.createMember)
	mux.HandleFunc("GET /api/members/{id}", a.getMember)
	mux.HandleFunc("PUT /api/members/{id}", a.updateMember)
	mux.HandleFunc("DELETE /api/members/{id}", a.deleteMember)

	a.registerSchedule(mux)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "endpoint not found")
	})
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if err := a.store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database tidak tersedia")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---- koloms ----

// MaxKolom bounds kolom numbers to something a congregation could have.
const MaxKolom = 999

func (a *API) listKoloms(w http.ResponseWriter, r *http.Request) {
	ks, err := a.store.ListKoloms(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ks)
}

func (a *API) createKolom(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Number int `json:"number"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Number < 1 || in.Number > MaxKolom {
		writeError(w, http.StatusBadRequest, "nomor kolom harus antara 1 dan 999")
		return
	}
	k, err := a.store.CreateKolom(r.Context(), in.Number)
	if errors.Is(err, store.ErrKolomExists) {
		writeError(w, http.StatusConflict, "kolom ini sudah ada")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, k)
}

func (a *API) deleteKolom(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	err := a.store.DeleteKolom(r.Context(), id)
	if errors.Is(err, store.ErrKolomInUse) {
		writeError(w, http.StatusConflict, "kolom masih memiliki keluarga atau jadwal ibadah; pindahkan atau hapus terlebih dahulu")
		return
	}
	if errors.Is(err, store.ErrKolomHasUsers) {
		writeError(w, http.StatusConflict, "kolom ini masih ditugaskan kepada pengguna; ubah kolom atau role pengguna itu terlebih dahulu")
		return
	}
	if err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- families ----

type familyInput struct {
	Kolom       int        `json:"kolom"`
	No          int        `json:"no"`
	Name        string     `json:"name"`
	Note        string     `json:"note"`
	Domicile    string     `json:"domicile"`
	WeddingDate *time.Time `json:"weddingDate"`
}

func (in familyInput) toModel() (model.Family, error) {
	f := model.Family{
		Kolom:       in.Kolom,
		No:          in.No,
		Name:        clean(in.Name),
		Note:        clean(in.Note),
		Domicile:    clean(in.Domicile),
		WeddingDate: dateOnly(in.WeddingDate),
	}
	if f.Kolom <= 0 {
		return f, errors.New("kolom wajib dipilih")
	}
	if f.Name == "" {
		return f, errors.New("nama keluarga wajib diisi")
	}
	return f, nil
}

func (a *API) listFamilies(w http.ResponseWriter, r *http.Request) {
	fams, err := a.store.ListFamilies(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	counts, err := a.store.MemberCounts(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	now := time.Now()
	full := dataView(w, r)
	out := make([]model.FamilyView, 0, len(fams))
	for _, f := range fams {
		v := model.NewFamilyView(f, counts[f.ID], now)
		if !seesPrivate(r, full, f.Kolom) {
			v = v.Public()
		}
		out = append(out, v)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getFamily(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	f, err := a.store.GetFamily(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	members, err := a.store.ListMembers(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}
	rules, err := a.store.BipraRules(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	now := time.Now()
	v := model.NewFamilyView(f, len(members), now)
	for _, m := range members {
		v.Members = append(v.Members, model.NewMemberView(m, f, now, rules))
	}
	if !seesPrivate(r, dataView(w, r), f.Kolom) {
		v = v.Public()
	}
	writeJSON(w, http.StatusOK, v)
}

func (a *API) createFamily(w http.ResponseWriter, r *http.Request) {
	var in familyInput
	if !decode(w, r, &in) {
		return
	}
	f, err := in.toModel()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !mayManage(w, r, f.Kolom) {
		return
	}
	f, err = a.store.CreateFamily(r.Context(), f)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, model.NewFamilyView(f, 0, time.Now()))
}

func (a *API) updateFamily(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in familyInput
	if !decode(w, r, &in) {
		return
	}
	f, err := in.toModel()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	f.ID = id
	// both the family's kolom and the one it would move to
	old, err := a.store.GetFamily(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	if !mayManage(w, r, old.Kolom) || !mayManage(w, r, f.Kolom) {
		return
	}
	f, err = a.store.UpdateFamily(r.Context(), f)
	if err != nil {
		storeError(w, err)
		return
	}
	n, err := a.store.CountMembers(r.Context(), f.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, model.NewFamilyView(f, n, time.Now()))
}

func (a *API) deleteFamily(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	old, err := a.store.GetFamily(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	if !mayManage(w, r, old.Kolom) {
		return
	}
	if err := a.store.DeleteFamily(r.Context(), id); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- members ----

type memberInput struct {
	FamilyID      string     `json:"familyId"`
	Order         int        `json:"order"`
	Name          string     `json:"name"`
	BirthPlace    string     `json:"birthPlace"`
	BirthDate     *time.Time `json:"birthDate"`
	Gender        string     `json:"gender"`
	MaritalStatus string     `json:"maritalStatus"`
	Relationship  string     `json:"relationship"`
	Baptized      *bool      `json:"baptized"`
	Confirmed     *bool      `json:"confirmed"`
}

func (in memberInput) toModel() (model.Member, error) {
	fid, err := bson.ObjectIDFromHex(in.FamilyID)
	if err != nil {
		return model.Member{}, errors.New("keluarga wajib dipilih")
	}
	m := model.Member{
		FamilyID:      fid,
		Order:         in.Order,
		Name:          clean(in.Name),
		BirthPlace:    clean(in.BirthPlace),
		BirthDate:     dateOnly(in.BirthDate),
		Gender:        clean(in.Gender),
		MaritalStatus: clean(in.MaritalStatus),
		Relationship:  clean(in.Relationship),
		Baptized:      in.Baptized,
		Confirmed:     in.Confirmed,
	}
	if m.Name == "" {
		return m, errors.New("nama lengkap wajib diisi")
	}
	if m.Gender != "" && m.Gender != model.GenderMale && m.Gender != model.GenderFemale {
		return m, errors.New("jenis kelamin tidak valid")
	}
	return m, nil
}

func (a *API) memberViews(r *http.Request, members []model.Member) ([]model.MemberView, error) {
	fams, err := a.store.ListFamilies(r.Context())
	if err != nil {
		return nil, err
	}
	byID := map[bson.ObjectID]model.Family{}
	for _, f := range fams {
		byID[f.ID] = f
	}
	rules, err := a.store.BipraRules(r.Context())
	if err != nil {
		return nil, err
	}
	now := time.Now()
	full := auth.SignedIn(r)
	own := ownBipra(r, rules)
	out := make([]model.MemberView, 0, len(members))
	for _, m := range members {
		f := byID[m.FamilyID]
		v := model.NewMemberView(m, f, now, rules)
		if !seesMember(r, full, f.Kolom, v, own) {
			v = v.Public()
		}
		out = append(out, v)
	}
	return out, nil
}

func (a *API) listMembers(w http.ResponseWriter, r *http.Request) {
	var familyID bson.ObjectID
	if q := r.URL.Query().Get("familyId"); q != "" {
		id, err := bson.ObjectIDFromHex(q)
		if err != nil {
			writeError(w, http.StatusBadRequest, "familyId tidak valid")
			return
		}
		familyID = id
	}
	members, err := a.store.ListMembers(r.Context(), familyID)
	if err != nil {
		serverError(w, err)
		return
	}
	dataView(w, r)
	out, err := a.memberViews(r, members)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getMember(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	m, err := a.store.GetMember(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	a.writeMember(w, r, http.StatusOK, m)
}

func (a *API) createMember(w http.ResponseWriter, r *http.Request) {
	var in memberInput
	if !decode(w, r, &in) {
		return
	}
	m, err := in.toModel()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !a.mayManageFamily(w, r, m.FamilyID) {
		return
	}
	m, err = a.store.CreateMember(r.Context(), m)
	if err != nil {
		storeError(w, err)
		return
	}
	a.writeMember(w, r, http.StatusCreated, m)
}

func (a *API) updateMember(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in memberInput
	if !decode(w, r, &in) {
		return
	}
	m, err := in.toModel()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	m.ID = id
	// both the member's family and the one it would move to
	old, err := a.store.GetMember(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	if !a.mayManageFamily(w, r, old.FamilyID) || !a.mayManageFamily(w, r, m.FamilyID) {
		return
	}
	m, err = a.store.UpdateMember(r.Context(), m)
	if err != nil {
		storeError(w, err)
		return
	}
	a.writeMember(w, r, http.StatusOK, m)
}

func (a *API) deleteMember(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	old, err := a.store.GetMember(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	if !a.mayManageFamily(w, r, old.FamilyID) {
		return
	}
	if err := a.store.DeleteMember(r.Context(), id); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) writeMember(w http.ResponseWriter, r *http.Request, status int, m model.Member) {
	f, err := a.store.GetFamily(r.Context(), m.FamilyID)
	if err != nil {
		storeError(w, err)
		return
	}
	rules, err := a.store.BipraRules(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	v := model.NewMemberView(m, f, time.Now(), rules)
	if !seesMember(r, dataView(w, r), f.Kolom, v, ownBipra(r, rules)) {
		v = v.Public()
	}
	writeJSON(w, status, v)
}

// ---- helpers ----

// mayManage reports whether the signed-in user may change data of kolom k
// (a "kelola data kolom" user only their own); if not, it answers 403.
func mayManage(w http.ResponseWriter, r *http.Request, k int) bool {
	u := auth.CurrentUser(r)
	if u == nil || u.ManagesKolom(k) { // writes always have a user (auth middleware)
		return true
	}
	writeError(w, http.StatusForbidden, u.Scope(), "forbidden")
	return false
}

// seesPrivate reports whether r gets the private data of kolom k: signed in,
// and (for a "kelola data kolom" user) their own kolom. full is dataView's answer.
func seesPrivate(r *http.Request, full bool, k int) bool {
	if !full {
		return false
	}
	u := auth.CurrentUser(r)
	return u == nil || u.ManagesKolom(k)
}

// dataView reports whether r gets the full (signed-in) view, and tells the
// client which one it is in the X-Data-View header ("full" or "public"), so
// the page never has to guess whether the data it holds is redacted.
func dataView(w http.ResponseWriter, r *http.Request) bool {
	full := auth.SignedIn(r)
	view := "public"
	if full {
		view = "full"
	}
	w.Header().Set("X-Data-View", view)
	return full
}

// ownBipra is the category of a "kelola data BIPRA" user (they see its
// members' private data), or nil.
func ownBipra(r *http.Request, rules model.BipraRules) *model.BipraCategory {
	u := auth.CurrentUser(r)
	if u == nil || u.Role != model.RoleBipra || u.BipraID == nil {
		return nil
	}
	for _, c := range rules {
		if c.ID == *u.BipraID {
			return &c
		}
	}
	return nil
}

// seesMember: seesPrivate for the member's kolom, or the member is in the
// category of a "kelola data BIPRA" user.
func seesMember(r *http.Request, full bool, kolom int, v model.MemberView, own *model.BipraCategory) bool {
	return seesPrivate(r, full, kolom) || (full && inBipra(v, own))
}

func inBipra(v model.MemberView, c *model.BipraCategory) bool {
	if c == nil {
		return false
	}
	if c.Kind == model.KindFlag {
		return slices.Contains(v.Flags, c.Key)
	}
	return v.Category == c.Key
}

// mayManageFamily is mayManage for the kolom of family id (a missing family
// is left to the store, which answers "not found").
func (a *API) mayManageFamily(w http.ResponseWriter, r *http.Request, id bson.ObjectID) bool {
	if u := auth.CurrentUser(r); u == nil || u.ManagesAll() {
		return true
	}
	f, err := a.store.GetFamily(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		return true
	}
	if err != nil {
		serverError(w, err)
		return false
	}
	return mayManage(w, r, f.Kolom)
}

func clean(s string) string { return strings.Join(strings.Fields(s), " ") }

func dateOnly(t *time.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return &d
}

func pathID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		return id, false
	}
	return id, true
}

func storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
		return
	}
	if errors.Is(err, store.ErrNoKolom) {
		writeError(w, http.StatusBadRequest, "kolom tidak ditemukan; tambahkan kolom terlebih dahulu")
		return
	}
	serverError(w, err)
}

var (
	writeJSON   = httpx.WriteJSON
	writeError  = httpx.WriteError
	serverError = httpx.ServerError
	decode      = httpx.Decode
)
