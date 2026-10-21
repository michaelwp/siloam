package api

import (
	"errors"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/model"
	"siloam/internal/store"
)

// ---- Kategorial BIPRA: the categories and their rules ----

// listBipra is public: the apps label and count members by these.
func (a *API) listBipra(w http.ResponseWriter, r *http.Request) {
	rules, err := a.store.BipraRules(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

type bipraInput struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Color     string `json:"color"`
	MinAge    *int   `json:"minAge"`
	MaxAge    *int   `json:"maxAge"`
	Gender    string `json:"gender"`
	Marital   string `json:"marital"`
	OrMarried bool   `json:"orMarried"`
}

func (in bipraInput) toModel() (model.BipraCategory, error) {
	c := model.BipraCategory{
		Key: in.Key, Name: in.Name, Kind: in.Kind, Color: in.Color,
		MinAge: in.MinAge, MaxAge: in.MaxAge, Gender: in.Gender, Marital: in.Marital, OrMarried: in.OrMarried,
	}
	return c, c.Normalize()
}

func bipraError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrBipraExists) {
		writeError(w, http.StatusConflict, "kode kategori ini sudah dipakai")
		return
	}
	if errors.Is(err, store.ErrBipraHasUsers) {
		writeError(w, http.StatusConflict, "kategori ini masih ditugaskan kepada pengguna; ubah role pengguna itu terlebih dahulu")
		return
	}
	storeError(w, err)
}

func (a *API) createBipra(w http.ResponseWriter, r *http.Request) {
	var in bipraInput
	if !decode(w, r, &in) {
		return
	}
	c, err := in.toModel()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if c, err = a.store.CreateBipra(r.Context(), c); err != nil {
		bipraError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (a *API) updateBipra(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in bipraInput
	if !decode(w, r, &in) {
		return
	}
	c, err := in.toModel()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	c.ID = id
	if c, err = a.store.UpdateBipra(r.Context(), c); err != nil {
		bipraError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *API) deleteBipra(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := a.store.DeleteBipra(r.Context(), id); err != nil {
		bipraError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// reorderBipra takes every category's id in the new order: the order of the
// lists, and which main category wins when several match.
func (a *API) reorderBipra(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs []string `json:"ids"`
	}
	if !decode(w, r, &in) {
		return
	}
	ids := make([]bson.ObjectID, 0, len(in.IDs))
	for _, s := range in.IDs {
		id, err := bson.ObjectIDFromHex(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "id tidak valid")
			return
		}
		ids = append(ids, id)
	}
	if err := a.store.ReorderBipra(r.Context(), ids); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "urutan harus memuat setiap kategori tepat satu kali; muat ulang halaman")
			return
		}
		serverError(w, err)
		return
	}
	a.listBipra(w, r)
}
