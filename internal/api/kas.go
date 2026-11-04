package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/auth"
	"siloam/internal/model"
	"siloam/internal/store"
)

// The kolom cash books ("Kas Kolom"). Admins and a kolom's treasurers keep
// it; who sees all of that kolom's data may read it.

func (a *API) registerKas(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/kas", a.listKas)
	mux.HandleFunc("GET /api/kas/summary", a.kasSummary)
	mux.HandleFunc("POST /api/kas", a.createKas)
	mux.HandleFunc("PUT /api/kas/{id}", a.updateKas)
	mux.HandleFunc("DELETE /api/kas/{id}", a.deleteKas)
}

// MaxKasAmount bounds one entry (Rp 10 trillion): a typo, not a real amount.
const MaxKasAmount = 10_000_000_000_000

// maxKasDescription is the longest description, in characters.
const maxKasDescription = 300

type kasInput struct {
	Kolom       int        `json:"kolom"`
	Kind        string     `json:"kind"`
	Date        *time.Time `json:"date"`
	Amount      int64      `json:"amount"`
	ServiceID   string     `json:"serviceId"` // "" = not from (or for) an ibadah
	Description string     `json:"description"`
}

// KasBook is a kolom's cash book: its entries (oldest first) and totals.
type KasBook struct {
	Kolom   int              `json:"kolom"`
	In      int64            `json:"in"`
	Out     int64            `json:"out"`
	Balance int64            `json:"balance"`
	Entries []model.KasEntry `json:"entries"`
	// CanEdit: the user may add, change and remove entries of this kolom
	CanEdit bool `json:"canEdit"`
}

func (a *API) listKas(w http.ResponseWriter, r *http.Request) {
	k, err := strconv.Atoi(r.URL.Query().Get("kolom"))
	if err != nil || k <= 0 {
		writeError(w, http.StatusBadRequest, "pilih kolom")
		return
	}
	u := auth.CurrentUser(r)
	if u == nil || !u.SeesKas(k) {
		writeError(w, http.StatusForbidden, "Anda tidak dapat melihat kas kolom ini", "forbidden")
		return
	}
	entries, err := a.store.ListKas(r.Context(), k)
	if err != nil {
		serverError(w, err)
		return
	}
	book := KasBook{Kolom: k, Entries: entries, CanEdit: u.ManagesKas(k)}
	for _, e := range entries {
		if e.Kind == model.KasIn {
			book.In += e.Amount
		} else {
			book.Out += e.Amount
		}
	}
	book.Balance = book.In - book.Out
	writeJSON(w, http.StatusOK, book)
}

// kasSummary is the totals of every kolom whose kas the user may read
// (koloms without entries included, with zeros), and of one month:
// ?month=YYYY-MM (the client's), default this month in the app's time zone.
func (a *API) kasSummary(w http.ResponseWriter, r *http.Request) {
	u := auth.CurrentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "silakan masuk terlebih dahulu", "unauthenticated")
		return
	}
	month := a.now().In(a.loc).Format("2006-01")
	if v := r.URL.Query().Get("month"); v != "" {
		month = v
	}
	// entry dates are midnight UTC of the day
	from, err := time.Parse("2006-01", month)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bulan tidak valid (YYYY-MM)")
		return
	}
	ks, err := a.store.ListKoloms(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	sums, err := a.store.KasSummaries(r.Context(), from, from.AddDate(0, 1, 0))
	if err != nil {
		serverError(w, err)
		return
	}
	byKolom := map[int]model.KasSummary{}
	for _, s := range sums {
		byKolom[s.Kolom] = s
	}
	out := []model.KasSummary{}
	for _, k := range ks {
		if !u.SeesKas(k.Number) {
			continue
		}
		s := byKolom[k.Number]
		s.Kolom = k.Number
		s.Month = month
		out = append(out, s)
	}
	writeJSON(w, http.StatusOK, out)
}

// kasEntry checks the input and makes the entry (with the linked ibadah's
// name). old is the entry being changed (nil: a new one): it may keep its
// link to an ibadah deleted since.
func (a *API) kasEntry(r *http.Request, in kasInput, old *model.KasEntry) (model.KasEntry, error) {
	e := model.KasEntry{Kolom: in.Kolom, Kind: in.Kind, Amount: in.Amount, Description: clean(in.Description)}
	if e.Kolom <= 0 {
		return e, bad("kolom wajib dipilih")
	}
	ok, err := a.store.KolomExists(r.Context(), e.Kolom)
	if err != nil {
		return e, err
	}
	if !ok {
		return e, bad("kolom %d tidak ada", e.Kolom)
	}
	if e.Kind != model.KasIn && e.Kind != model.KasOut {
		return e, bad("pilih kas masuk atau kas keluar")
	}
	if e.Amount <= 0 {
		return e, bad("jumlah harus lebih dari 0")
	}
	if e.Amount > MaxKasAmount {
		return e, bad("jumlah terlalu besar")
	}
	d := dateOnly(in.Date)
	if d == nil {
		return e, bad("tanggal wajib diisi")
	}
	e.Date = *d
	if utf8.RuneCountInString(e.Description) > maxKasDescription {
		return e, bad("keterangan paling banyak %d karakter", maxKasDescription)
	}
	if in.ServiceID == "" {
		if e.Description == "" {
			if e.Kind == model.KasIn {
				return e, bad("tulis keterangan sumber kas (atau pilih ibadahnya)")
			}
			return e, bad("tulis keterangan penggunaan kas (atau pilih ibadahnya)")
		}
		return e, nil
	}
	id, err := bson.ObjectIDFromHex(in.ServiceID)
	if err != nil {
		return e, bad("ibadah tidak valid")
	}
	sv, err := a.store.GetService(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		if old != nil && old.ServiceID != nil && *old.ServiceID == id && old.Kolom == e.Kolom {
			e.ServiceID, e.ServiceLabel = &id, old.ServiceLabel
			return e, nil
		}
		return e, bad("ibadah tidak ditemukan")
	}
	if err != nil {
		return e, err
	}
	if sv.Kolom != e.Kolom {
		return e, bad("ibadah itu bukan ibadah Kolom %d", e.Kolom)
	}
	t, _ := a.store.GetWorshipType(r.Context(), sv.TypeID) // a missing jenis only shortens the name
	e.ServiceID = &id
	e.ServiceLabel = model.NewServiceView(sv, t, a.loc).Title() + " · " + sv.Date.Format("2006-01-02")
	return e, nil
}

// inputError is a refused kas input (400); other errors are the server's.
type inputError string

func (e inputError) Error() string { return string(e) }

func bad(format string, args ...any) error { return inputError(fmt.Sprintf(format, args...)) }

// kasFail answers err from kasEntry.
func kasFail(w http.ResponseWriter, err error) {
	var ie inputError
	if errors.As(err, &ie) {
		writeError(w, http.StatusBadRequest, ie.Error())
		return
	}
	serverError(w, err)
}

func (a *API) mayKeepKas(w http.ResponseWriter, r *http.Request, k int) bool {
	u := auth.CurrentUser(r)
	if u != nil && u.ManagesKas(k) {
		return true
	}
	msg := "hanya admin dan bendahara kolom yang dapat mengubah kas"
	if u != nil && u.IsTreasurer() {
		msg = u.Scope()
	}
	writeError(w, http.StatusForbidden, msg, "forbidden")
	return false
}

func (a *API) createKas(w http.ResponseWriter, r *http.Request) {
	var in kasInput
	if !decode(w, r, &in) {
		return
	}
	if in.Kolom > 0 && !a.mayKeepKas(w, r, in.Kolom) {
		return
	}
	e, err := a.kasEntry(r, in, nil)
	if err != nil {
		kasFail(w, err)
		return
	}
	e.CreatedBy = username(r)
	e, err = a.store.CreateKas(r.Context(), e)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (a *API) updateKas(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in kasInput
	if !decode(w, r, &in) {
		return
	}
	old, err := a.store.GetKas(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	// both the kolom it was in and the one it moves to
	if !a.mayKeepKas(w, r, old.Kolom) || (in.Kolom > 0 && in.Kolom != old.Kolom && !a.mayKeepKas(w, r, in.Kolom)) {
		return
	}
	e, err := a.kasEntry(r, in, &old)
	if err != nil {
		kasFail(w, err)
		return
	}
	e.ID = id
	e.UpdatedBy = username(r)
	e, err = a.store.UpdateKas(r.Context(), e)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (a *API) deleteKas(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	old, err := a.store.GetKas(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	if !a.mayKeepKas(w, r, old.Kolom) {
		return
	}
	if err := a.store.DeleteKas(r.Context(), id); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func username(r *http.Request) string {
	if u := auth.CurrentUser(r); u != nil {
		return u.Username
	}
	return ""
}
