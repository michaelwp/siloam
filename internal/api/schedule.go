package api

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/ics"
	"siloam/internal/model"
	"siloam/internal/store"
)

func (a *API) registerSchedule(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/worship-types", a.listWorshipTypes)
	mux.HandleFunc("POST /api/worship-types", a.createWorshipType)
	mux.HandleFunc("PUT /api/worship-types/{id}", a.updateWorshipType)
	mux.HandleFunc("DELETE /api/worship-types/{id}", a.deleteWorshipType)

	mux.HandleFunc("GET /api/services", a.listServices)
	mux.HandleFunc("POST /api/services", a.createService)
	mux.HandleFunc("PUT /api/services/{id}", a.updateService)
	mux.HandleFunc("DELETE /api/services/{id}", a.deleteService)

	// iCalendar: one service (Apple Calendar "add"), or a feed to subscribe to
	mux.HandleFunc("GET /api/services/{id}/ics", a.serviceICS)
	mux.HandleFunc("GET /api/calendar.ics", a.calendarICS)
}

// ---- worship types ----

var colorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type worshipTypeInput struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (in worshipTypeInput) toModel() (model.WorshipType, error) {
	t := model.WorshipType{Name: clean(in.Name), Color: strings.ToLower(in.Color)}
	if t.Name == "" {
		return t, errors.New("nama jenis ibadah wajib diisi")
	}
	if len(t.Name) > 80 {
		return t, errors.New("nama jenis ibadah terlalu panjang")
	}
	if !colorRe.MatchString(t.Color) {
		return t, errors.New("warna tidak valid")
	}
	return t, nil
}

func (a *API) listWorshipTypes(w http.ResponseWriter, r *http.Request) {
	ts, err := a.store.ListWorshipTypes(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ts)
}

func (a *API) createWorshipType(w http.ResponseWriter, r *http.Request) {
	var in worshipTypeInput
	if !decode(w, r, &in) {
		return
	}
	t, err := in.toModel()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err = a.store.CreateWorshipType(r.Context(), t)
	if err != nil {
		scheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (a *API) updateWorshipType(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in worshipTypeInput
	if !decode(w, r, &in) {
		return
	}
	t, err := in.toModel()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t.ID = id
	t, err = a.store.UpdateWorshipType(r.Context(), t)
	if err != nil {
		scheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *API) deleteWorshipType(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := a.store.DeleteWorshipType(r.Context(), id); err != nil {
		scheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- services ----

var timeRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

type serviceInput struct {
	Kolom    int        `json:"kolom"`
	TypeID   string     `json:"typeId"`
	Date     *time.Time `json:"date"`
	Time     string     `json:"time"`
	Host     string     `json:"host"`
	Location string     `json:"location"`
	Leader   string     `json:"leader"`
	LeaderID string     `json:"leaderId"`
	MC       string     `json:"mc"`
	MCID     string     `json:"mcId"`
	Note     string     `json:"note"`
}

// optionalID parses an optional member ID ("" or null → nil).
func optionalID(v, what string) (*bson.ObjectID, error) {
	if v == "" {
		return nil, nil
	}
	id, err := bson.ObjectIDFromHex(v)
	if err != nil {
		return nil, fmt.Errorf("%s tidak valid", what)
	}
	return &id, nil
}

func (in serviceInput) toModel() (model.Service, error) {
	sv := model.Service{
		Kolom:    in.Kolom,
		Time:     strings.TrimSpace(in.Time),
		Host:     strings.TrimSpace(strings.TrimPrefix(clean(in.Host), "Kel. ")),
		Location: clean(in.Location),
		Leader:   clean(in.Leader),
		MC:       clean(in.MC),
		Note:     strings.TrimSpace(in.Note),
	}
	if sv.Kolom <= 0 {
		return sv, errors.New("kolom wajib dipilih")
	}
	tid, err := bson.ObjectIDFromHex(in.TypeID)
	if err != nil {
		return sv, errors.New("jenis ibadah wajib dipilih")
	}
	sv.TypeID = tid
	d := dateOnly(in.Date)
	if d == nil {
		return sv, errors.New("tanggal wajib diisi")
	}
	sv.Date = *d
	if !timeRe.MatchString(sv.Time) {
		return sv, errors.New("jam tidak valid (gunakan format 19:00)")
	}
	if len(sv.Note) > 2000 {
		return sv, errors.New("keterangan terlalu panjang")
	}
	if sv.LeaderID, err = optionalID(in.LeaderID, "pemimpin ibadah"); err != nil {
		return sv, err
	}
	if sv.MCID, err = optionalID(in.MCID, "pembawa acara"); err != nil {
		return sv, err
	}
	return sv, nil
}

// linkPeople checks that linked members exist and takes their names, so the
// stored name matches the member at the time of saving.
func (a *API) linkPeople(r *http.Request, sv *model.Service) error {
	for _, p := range []struct {
		id   *bson.ObjectID
		name *string
		what string
	}{{sv.LeaderID, &sv.Leader, "pemimpin ibadah"}, {sv.MCID, &sv.MC, "pembawa acara"}} {
		if p.id == nil {
			continue
		}
		m, err := a.store.GetMember(r.Context(), *p.id)
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("%s: anggota tidak ditemukan", p.what)
		}
		if err != nil {
			return err
		}
		*p.name = m.Name
	}
	return nil
}

func (a *API) serviceViews(r *http.Request, svs []model.Service) ([]model.ServiceView, error) {
	ts, err := a.store.ListWorshipTypes(r.Context())
	if err != nil {
		return nil, err
	}
	byID := map[bson.ObjectID]model.WorshipType{}
	for _, t := range ts {
		byID[t.ID] = t
	}
	names, err := a.linkedNames(r, svs)
	if err != nil {
		return nil, err
	}
	out := make([]model.ServiceView, 0, len(svs))
	for _, sv := range svs {
		// linked people show their current name (renamed members follow)
		if sv.LeaderID != nil && names[*sv.LeaderID] != "" {
			sv.Leader = names[*sv.LeaderID]
		}
		if sv.MCID != nil && names[*sv.MCID] != "" {
			sv.MC = names[*sv.MCID]
		}
		out = append(out, model.NewServiceView(sv, byID[sv.TypeID], a.loc))
	}
	return out, nil
}

// linkedNames returns the current names of the members the services link to.
func (a *API) linkedNames(r *http.Request, svs []model.Service) (map[bson.ObjectID]string, error) {
	var ids []bson.ObjectID
	for _, sv := range svs {
		for _, id := range []*bson.ObjectID{sv.LeaderID, sv.MCID} {
			if id != nil {
				ids = append(ids, *id)
			}
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return a.store.MemberNames(r.Context(), ids)
}

func (a *API) listServices(w http.ResponseWriter, r *http.Request) {
	svs, err := a.store.ListServices(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	out, err := a.serviceViews(r, svs)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) writeService(w http.ResponseWriter, r *http.Request, status int, sv model.Service) {
	out, err := a.serviceViews(r, []model.Service{sv})
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, status, out[0])
}

func (a *API) createService(w http.ResponseWriter, r *http.Request) {
	var in serviceInput
	if !decode(w, r, &in) {
		return
	}
	sv, err := in.toModel()
	if err == nil {
		err = a.linkPeople(r, &sv)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sv, err = a.store.CreateService(r.Context(), sv)
	if err != nil {
		scheduleError(w, err)
		return
	}
	a.writeService(w, r, http.StatusCreated, sv)
}

func (a *API) updateService(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in serviceInput
	if !decode(w, r, &in) {
		return
	}
	sv, err := in.toModel()
	if err == nil {
		err = a.linkPeople(r, &sv)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sv.ID = id
	sv, err = a.store.UpdateService(r.Context(), sv)
	if err != nil {
		scheduleError(w, err)
		return
	}
	a.writeService(w, r, http.StatusOK, sv)
}

func (a *API) deleteService(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := a.store.DeleteService(r.Context(), id); err != nil {
		scheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- iCalendar ----

func (a *API) serviceICS(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	sv, err := a.store.GetService(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	views, err := a.serviceViews(r, []model.Service{sv})
	if err != nil {
		serverError(w, err)
		return
	}
	v := views[0]
	writeICS(w, fmt.Sprintf("ibadah-%s.ics", v.Start.Format("2006-01-02")), true,
		ics.Calendar("Jadwal Ibadah GMIM Siloam", []ics.Event{icsEvent(v)}))
}

// calendarICS is a subscribable feed of the schedule, optionally narrowed with
// ?kolom=11 and ?type=<id> (repeatable).
func (a *API) calendarICS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	kolom := 0
	if k := q.Get("kolom"); k != "" {
		n, err := strconv.Atoi(k)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "kolom tidak valid")
			return
		}
		kolom = n
	}
	types := map[string]bool{}
	for _, t := range q["type"] {
		types[t] = true
	}
	svs, err := a.store.ListServices(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	views, err := a.serviceViews(r, svs)
	if err != nil {
		serverError(w, err)
		return
	}
	events := make([]ics.Event, 0, len(views))
	for _, v := range views {
		if (kolom == 0 || v.Kolom == kolom) && (len(types) == 0 || types[v.TypeID.Hex()]) {
			events = append(events, icsEvent(v))
		}
	}
	name := "Jadwal Ibadah GMIM Siloam"
	if kolom > 0 {
		name += fmt.Sprintf(" · Kolom %d", kolom)
	}
	writeICS(w, "jadwal-ibadah.ics", false, ics.Calendar(name, events))
}

func icsEvent(v model.ServiceView) ics.Event {
	var desc []string
	desc = append(desc, fmt.Sprintf("Kolom %d", v.Kolom))
	if v.Leader != "" {
		desc = append(desc, "Pemimpin Ibadah: "+v.Leader)
	}
	if v.MC != "" {
		desc = append(desc, "Pembawa Acara: "+v.MC)
	}
	if v.Note != "" {
		desc = append(desc, v.Note)
	}
	loc := v.Location
	if loc == "" && v.Host != "" {
		loc = "Rumah Kel. " + v.Host
	}
	return ics.Event{
		UID:         v.ID.Hex() + "@gmim-siloam",
		Start:       v.Start,
		End:         v.End,
		Summary:     v.Title(),
		Location:    loc,
		Description: strings.Join(desc, "\n"),
		Updated:     v.UpdatedAt,
	}
}

func writeICS(w http.ResponseWriter, filename string, download bool, body []byte) {
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	disp := "inline"
	if download {
		disp = "attachment"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disp, filename))
	_, _ = w.Write(body)
}

func scheduleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrTypeExists):
		writeError(w, http.StatusConflict, "jenis ibadah dengan nama ini sudah ada")
	case errors.Is(err, store.ErrTypeInUse):
		writeError(w, http.StatusConflict, "jenis ibadah ini masih dipakai di jadwal; ubah atau hapus jadwalnya terlebih dahulu")
	case errors.Is(err, store.ErrNoType):
		writeError(w, http.StatusBadRequest, "jenis ibadah tidak ditemukan")
	default:
		storeError(w, err)
	}
}
