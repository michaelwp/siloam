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

// The cash books: each kolom's ("Kas Kolom") and each BIPRA category's
// ("Kas BIPRA", for the whole congregation). Admins and the book's
// treasurers keep it; see User.SeesKas and User.SeesBipraKas for who reads it.

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
	Kolom       int        `json:"kolom"`   // a kolom's book, or
	BipraID     string     `json:"bipraId"` // a BIPRA category's
	Kind        string     `json:"kind"`
	Date        *time.Time `json:"date"`
	Amount      int64      `json:"amount"`
	ServiceID   string     `json:"serviceId"` // "" = not from (or for) an ibadah
	Description string     `json:"description"`
}

// KasBook is a cash book (a kolom's, or a BIPRA category's: BipraID and
// Name): its entries (oldest first) and totals.
type KasBook struct {
	Kolom   int              `json:"kolom"`
	BipraID *bson.ObjectID   `json:"bipraId,omitempty"`
	Name    string           `json:"name,omitempty"`
	In      int64            `json:"in"`
	Out     int64            `json:"out"`
	Balance int64            `json:"balance"`
	Entries []model.KasEntry `json:"entries"`
	// CanEdit: the user may add, change and remove entries of this book
	CanEdit bool `json:"canEdit"`
}

// listKas is one book: ?kolom=N or ?bipra=<category id>.
func (a *API) listKas(w http.ResponseWriter, r *http.Request) {
	u := auth.CurrentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "silakan masuk terlebih dahulu", "unauthenticated")
		return
	}
	var book KasBook
	var entries []model.KasEntry
	var err error
	if v := r.URL.Query().Get("bipra"); v != "" {
		id, perr := bson.ObjectIDFromHex(v)
		if perr != nil {
			writeError(w, http.StatusBadRequest, "kategori BIPRA tidak valid")
			return
		}
		if !u.SeesBipraKas(id) {
			writeError(w, http.StatusForbidden, "Anda tidak dapat melihat kas kategori ini", "forbidden")
			return
		}
		c, cerr := a.store.BipraCategory(r.Context(), id)
		if cerr != nil {
			storeError(w, cerr)
			return
		}
		book = KasBook{BipraID: &id, Name: c.Name, CanEdit: u.ManagesBipraKas(id)}
		entries, err = a.store.ListBipraKas(r.Context(), id)
	} else {
		k, perr := strconv.Atoi(r.URL.Query().Get("kolom"))
		if perr != nil || k <= 0 {
			writeError(w, http.StatusBadRequest, "pilih kolom atau kategori BIPRA")
			return
		}
		if !u.SeesKas(k) {
			writeError(w, http.StatusForbidden, "Anda tidak dapat melihat kas kolom ini", "forbidden")
			return
		}
		book = KasBook{Kolom: k, CanEdit: u.ManagesKas(k)}
		entries, err = a.store.ListKas(r.Context(), k)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	book.Entries = entries
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
// (koloms without entries included, with zeros), or with ?book=bipra of
// every BIPRA category (in their order), and of one month: ?month=YYYY-MM
// (the client's), default this month in the app's time zone.
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
	to := from.AddDate(0, 1, 0)
	out := []model.KasSummary{}
	if r.URL.Query().Get("book") == "bipra" {
		cats, err := a.store.BipraRules(r.Context())
		if err != nil {
			serverError(w, err)
			return
		}
		sums, err := a.store.KasBipraSummaries(r.Context(), from, to)
		if err != nil {
			serverError(w, err)
			return
		}
		byID := map[bson.ObjectID]model.KasSummary{}
		for _, s := range sums {
			byID[*s.BipraID] = s
		}
		for _, c := range cats {
			if !u.SeesBipraKas(c.ID) {
				continue
			}
			s := byID[c.ID]
			id := c.ID
			s.BipraID, s.Name, s.Month = &id, c.Name, month
			out = append(out, s)
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	ks, err := a.store.ListKoloms(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	sums, err := a.store.KasSummaries(r.Context(), from, to)
	if err != nil {
		serverError(w, err)
		return
	}
	byKolom := map[int]model.KasSummary{}
	for _, s := range sums {
		byKolom[s.Kolom] = s
	}
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
// name, and the book's name in Book). old is the entry being changed (nil:
// a new one): it may keep its link to an ibadah deleted since.
func (a *API) kasEntry(r *http.Request, in kasInput, old *model.KasEntry) (model.KasEntry, error) {
	e := model.KasEntry{Kind: in.Kind, Amount: in.Amount, Description: clean(in.Description)}
	switch {
	case in.BipraID != "" && in.Kolom != 0:
		return e, bad("pilih kolom atau kategori BIPRA, bukan keduanya")
	case in.BipraID != "":
		id, err := bson.ObjectIDFromHex(in.BipraID)
		if err != nil {
			return e, bad("kategori BIPRA tidak valid")
		}
		c, err := a.store.BipraCategory(r.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			return e, bad("kategori BIPRA tidak ada")
		}
		if err != nil {
			return e, err
		}
		e.BipraID, e.Book = &id, model.BipraBook(c.Name)
	case in.Kolom > 0:
		ok, err := a.store.KolomExists(r.Context(), in.Kolom)
		if err != nil {
			return e, err
		}
		if !ok {
			return e, bad("kolom %d tidak ada", in.Kolom)
		}
		e.Kolom, e.Book = in.Kolom, model.KolomBook(in.Kolom)
	default:
		return e, bad("kolom atau kategori BIPRA wajib dipilih")
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
		if old != nil && old.ServiceID != nil && *old.ServiceID == id && sameBook(*old, e) {
			e.ServiceID, e.ServiceLabel = &id, old.ServiceLabel
			return e, nil
		}
		return e, bad("ibadah tidak ditemukan")
	}
	if err != nil {
		return e, err
	}
	t, terr := a.store.GetWorshipType(r.Context(), sv.TypeID)
	if terr != nil && !errors.Is(terr, store.ErrNoType) {
		return e, terr
	}
	// a kolom's ibadah, or (any kolom) one of the jenis linked to the category
	if e.BipraID != nil {
		if t.BipraID == nil || *t.BipraID != *e.BipraID {
			return e, bad("ibadah itu bukan ibadah kategori ini")
		}
	} else if sv.Kolom != e.Kolom {
		return e, bad("ibadah itu bukan ibadah Kolom %d", e.Kolom)
	}
	e.ServiceID = &id
	e.ServiceLabel = model.NewServiceView(sv, t, a.loc).Title() + " · " + sv.Date.Format("2006-01-02")
	return e, nil
}

// sameBook reports whether a and b are in the same cash book.
func sameBook(a, b model.KasEntry) bool {
	if a.BipraID != nil || b.BipraID != nil {
		return a.BipraID != nil && b.BipraID != nil && *a.BipraID == *b.BipraID
	}
	return a.Kolom == b.Kolom
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

// mayKeepKas reports whether the user may change the book of e (its kolom
// or category); if not, it answers 403.
func (a *API) mayKeepKas(w http.ResponseWriter, r *http.Request, e model.KasEntry) bool {
	u := auth.CurrentUser(r)
	if u != nil && ((e.BipraID != nil && u.ManagesBipraKas(*e.BipraID)) || (e.BipraID == nil && u.ManagesKas(e.Kolom))) {
		return true
	}
	msg := "hanya admin dan bendahara yang dapat mengubah kas"
	if u != nil && u.IsTreasurer() {
		msg = u.Scope()
	}
	writeError(w, http.StatusForbidden, msg, "forbidden")
	return false
}

// target is the book an input names (unchecked: kasEntry checks it).
func target(in kasInput) model.KasEntry {
	e := model.KasEntry{Kolom: in.Kolom}
	if id, err := bson.ObjectIDFromHex(in.BipraID); err == nil {
		e.BipraID = &id
	}
	return e
}

func (a *API) createKas(w http.ResponseWriter, r *http.Request) {
	var in kasInput
	if !decode(w, r, &in) {
		return
	}
	if (in.Kolom > 0 || in.BipraID != "") && !a.mayKeepKas(w, r, target(in)) {
		return
	}
	e, err := a.kasEntry(r, in, nil)
	if err != nil {
		kasFail(w, err)
		return
	}
	e.CreatedBy = username(r)
	book := e.Book
	e, err = a.store.CreateKas(r.Context(), e)
	if err != nil {
		serverError(w, err)
		return
	}
	e.Book = book
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
	// both the book it was in and the one it moves to
	to := target(in)
	if !a.mayKeepKas(w, r, old) || ((in.Kolom > 0 || in.BipraID != "") && !sameBook(old, to) && !a.mayKeepKas(w, r, to)) {
		return
	}
	e, err := a.kasEntry(r, in, &old)
	if err != nil {
		kasFail(w, err)
		return
	}
	e.ID = id
	e.UpdatedBy = username(r)
	book := e.Book
	e, err = a.store.UpdateKas(r.Context(), e)
	if err != nil {
		storeError(w, err)
		return
	}
	e.Book = book
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
	if !a.mayKeepKas(w, r, old) {
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
