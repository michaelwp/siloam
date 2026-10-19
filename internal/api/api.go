// Package api exposes the REST endpoints under /api.
package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/httpx"
	"siloam/internal/model"
	"siloam/internal/store"
)

type API struct {
	store *store.Store
}

func New(s *store.Store) *API { return &API{store: s} }

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", a.health)

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

// ---- families ----

type familyInput struct {
	No          int        `json:"no"`
	Name        string     `json:"name"`
	Note        string     `json:"note"`
	Domicile    string     `json:"domicile"`
	WeddingDate *time.Time `json:"weddingDate"`
}

func (in familyInput) toModel() (model.Family, error) {
	f := model.Family{
		No:          in.No,
		Name:        clean(in.Name),
		Note:        clean(in.Note),
		Domicile:    clean(in.Domicile),
		WeddingDate: dateOnly(in.WeddingDate),
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
	out := make([]model.FamilyView, 0, len(fams))
	for _, f := range fams {
		out = append(out, model.NewFamilyView(f, counts[f.ID], now))
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
	v := model.NewFamilyView(f, len(members), time.Now())
	v.Members = members
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
	f, err = a.store.CreateFamily(r.Context(), f)
	if err != nil {
		serverError(w, err)
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
	names := map[bson.ObjectID]string{}
	for _, f := range fams {
		names[f.ID] = f.Name
	}
	now := time.Now()
	out := make([]model.MemberView, 0, len(members))
	for _, m := range members {
		out = append(out, model.NewMemberView(m, names[m.FamilyID], now))
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
	writeJSON(w, status, model.NewMemberView(m, f.Name, time.Now()))
}

// ---- helpers ----

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
	serverError(w, err)
}

var (
	writeJSON   = httpx.WriteJSON
	writeError  = httpx.WriteError
	serverError = httpx.ServerError
	decode      = httpx.Decode
)
