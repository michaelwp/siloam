package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/auth"
	"siloam/internal/model"
	"siloam/internal/store"
)

// testAPI needs a MongoDB (MONGODB_TEST_URI, default local) and uses a
// throwaway database; skipped when none is reachable.
func testAPI(t *testing.T) (*API, *store.Store, http.Handler) {
	t.Helper()
	base := os.Getenv("MONGODB_TEST_URI")
	if base == "" {
		base = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	st, err := store.Open(ctx, strings.TrimRight(base, "/")+"/siloam_test_"+bson.NewObjectID().Hex(), "")
	if err != nil {
		t.Skipf("no MongoDB: %v", err)
	}
	t.Cleanup(func() {
		_ = st.DropDatabase(context.Background())
		_ = st.Close(context.Background())
	})
	if err := st.SeedWorshipTypes(context.Background()); err != nil {
		t.Fatal(err)
	}
	a := New(st, time.UTC)
	// a fixed today, before the services the tests add (dated today or earlier is locked)
	a.now = func() time.Time { return testToday.Add(12 * time.Hour) }
	mux := http.NewServeMux()
	a.Register(mux)
	return a, st, mux
}

var testToday = time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)

func TestServicesTodayOrEarlierAreLocked(t *testing.T) {
	_, st, h := testAPI(t)
	ctx := context.Background()
	types, _ := st.ListWorshipTypes(ctx)
	admin := &model.User{ID: bson.NewObjectID(), Username: "admin", Role: model.RoleAdmin}
	body := func(date string) string {
		return `{"kolom":3,"typeId":"` + types[0].ID.Hex() + `","date":"` + date + `T00:00:00Z","time":"19:00"}`
	}
	past, _ := st.CreateService(ctx, model.Service{Kolom: 3, TypeID: types[0].ID, Date: testToday.AddDate(0, 0, -3), Time: "19:00"})
	today, _ := st.CreateService(ctx, model.Service{Kolom: 3, TypeID: types[0].ID, Date: testToday, Time: "19:00"})
	future, _ := st.CreateService(ctx, model.Service{Kolom: 3, TypeID: types[0].ID, Date: testToday.AddDate(0, 0, 1), Time: "19:00"})

	for _, c := range []struct {
		name, method, path, body string
		want                     int
	}{
		{"add yesterday", "POST", "/api/services", body("2026-10-21"), http.StatusBadRequest},
		{"add today", "POST", "/api/services", body("2026-10-22"), http.StatusBadRequest},
		{"add tomorrow", "POST", "/api/services", body("2026-10-23"), http.StatusCreated},
		{"edit a past one", "PUT", "/api/services/" + past.ID.Hex(), body("2026-11-01"), http.StatusConflict},
		{"edit today's", "PUT", "/api/services/" + today.ID.Hex(), body("2026-11-01"), http.StatusConflict},
		{"move a future one to today", "PUT", "/api/services/" + future.ID.Hex(), body("2026-10-22"), http.StatusBadRequest},
		{"edit a future one", "PUT", "/api/services/" + future.ID.Hex(), body("2026-10-24"), http.StatusOK},
		{"delete a past one", "DELETE", "/api/services/" + past.ID.Hex(), "", http.StatusConflict},
		{"delete today's", "DELETE", "/api/services/" + today.ID.Hex(), "", http.StatusConflict},
		{"delete a future one", "DELETE", "/api/services/" + future.ID.Hex(), "", http.StatusNoContent},
	} {
		if got, out := call(h, admin, c.method, c.path, c.body); got != c.want {
			t.Errorf("%s: %d, want %d (%v)", c.name, got, c.want, out)
		}
	}
	if n, _ := st.ListServices(ctx); len(n) != 3 { // past, today and the one added for tomorrow
		t.Errorf("%d services left, want 3", len(n))
	}
}

// call sends a request as u and returns the status and the decoded body.
func call(h http.Handler, u *model.User, method, path, body string) (int, map[string]any) {
	r := auth.WithUser(httptest.NewRequest(method, path, strings.NewReader(body)), u)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestKolomUserManagesOnlyTheirKolom(t *testing.T) {
	_, st, h := testAPI(t)
	ctx := context.Background()
	own, err := st.CreateFamily(ctx, model.Family{Kolom: 3, Name: "Umboh"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := st.CreateFamily(ctx, model.Family{Kolom: 4, Name: "Kowaas"})
	if err != nil {
		t.Fatal(err)
	}
	born := time.Date(1990, 5, 6, 0, 0, 0, 0, time.UTC)
	otherMember, err := st.CreateMember(ctx, model.Member{FamilyID: other.ID, Name: "Ani Kowaas", BirthDate: &born})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateMember(ctx, model.Member{FamilyID: own.ID, Name: "Budi Umboh", BirthDate: &born}); err != nil {
		t.Fatal(err)
	}
	types, err := st.ListWorshipTypes(ctx)
	if err != nil || len(types) == 0 {
		t.Fatal(types, err)
	}
	day := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	otherService, err := st.CreateService(ctx, model.Service{Kolom: 4, TypeID: types[0].ID, Date: day, Time: "19:00"})
	if err != nil {
		t.Fatal(err)
	}

	k3 := &model.User{ID: bson.NewObjectID(), Username: "kolom3", Role: model.RoleKolom, Kolom: 3}
	family := func(kolom int) string { return `{"kolom":` + map[int]string{3: "3", 4: "4"}[kolom] + `,"name":"Baru"}` }
	member := func(f bson.ObjectID) string { return `{"familyId":"` + f.Hex() + `","name":"Anak Baru"}` }
	service := func(kolom string) string {
		return `{"kolom":` + kolom + `,"typeId":"` + types[0].ID.Hex() + `","date":"2026-11-11T00:00:00Z","time":"19:00"}`
	}
	for _, c := range []struct {
		what, method, path, body string
		want                     int
	}{
		{"add a family to their kolom", "POST", "/api/families", family(3), 201},
		{"add a family to another kolom", "POST", "/api/families", family(4), 403},
		{"move their family to another kolom", "PUT", "/api/families/" + own.ID.Hex(), family(4), 403},
		{"edit another kolom's family", "PUT", "/api/families/" + other.ID.Hex(), family(4), 403},
		{"delete another kolom's family", "DELETE", "/api/families/" + other.ID.Hex(), "", 403},
		{"add a member to their family", "POST", "/api/members", member(own.ID), 201},
		{"add a member to another kolom's family", "POST", "/api/members", member(other.ID), 403},
		{"move a member out of their kolom", "PUT", "/api/members/" + otherMember.ID.Hex(), member(own.ID), 403},
		{"delete another kolom's member", "DELETE", "/api/members/" + otherMember.ID.Hex(), "", 403},
		{"add a service to their kolom", "POST", "/api/services", service("3"), 201},
		{"add a service to another kolom", "POST", "/api/services", service("4"), 403},
		{"delete another kolom's service", "DELETE", "/api/services/" + otherService.ID.Hex(), "", 403},
	} {
		if got, body := call(h, k3, c.method, c.path, c.body); got != c.want {
			t.Errorf("%s: %d, want %d (%v)", c.what, got, c.want, body)
		}
	}

	// a regular user ("Pengguna") still manages every kolom
	staff := &model.User{ID: bson.NewObjectID(), Username: "staf", Role: model.RoleUser}
	if got, body := call(h, staff, "POST", "/api/families", family(4)); got != 201 {
		t.Errorf("regular user, another kolom: %d (%v)", got, body)
	}

	// private data: full for their kolom, the public view for the others
	r := auth.WithUser(httptest.NewRequest(http.MethodGet, "/api/members", nil), k3)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	var members []model.MemberView
	if err := json.Unmarshal(rec.Body.Bytes(), &members); err != nil {
		t.Fatal(err)
	}
	for _, m := range members {
		switch m.Name {
		case "Budi Umboh":
			if m.BirthDate == nil {
				t.Error("their own kolom's member is redacted")
			}
		case "Ani Kowaas":
			if m.BirthDate != nil {
				t.Error("another kolom's birth date is visible")
			}
		}
	}
}

func TestBipraRulesDecideMemberCategories(t *testing.T) {
	_, st, h := testAPI(t)
	ctx := context.Background()
	if err := st.SeedBipra(ctx); err != nil {
		t.Fatal(err)
	}
	f, err := st.CreateFamily(ctx, model.Family{Kolom: 3, Name: "Umboh"})
	if err != nil {
		t.Fatal(err)
	}
	born := time.Now().AddDate(-20, 0, -1) // 20 years old
	if _, err := st.CreateMember(ctx, model.Member{FamilyID: f.ID, Name: "Sari", Gender: model.GenderFemale, MaritalStatus: model.StatusSingle, BirthDate: &born}); err != nil {
		t.Fatal(err)
	}
	admin := &model.User{ID: bson.NewObjectID(), Username: "admin", Role: model.RoleAdmin}
	category := func() (string, []any) {
		r := auth.WithUser(httptest.NewRequest(http.MethodGet, "/api/members", nil), admin)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		var ms []map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &ms)
		flags, _ := ms[0]["flags"].([]any)
		return ms[0]["category"].(string), flags
	}
	if got, _ := category(); got != "Pemuda" {
		t.Fatalf("default: %s", got)
	}

	// a new main category for young women, then moved above Pemuda
	code, body := call(h, admin, http.MethodPost, "/api/admin/bipra",
		`{"key":"Pemudi","name":"Pemudi","kind":"main","color":"#8e24aa","minAge":17,"maxAge":30,"gender":"Perempuan","marital":"unmarried"}`)
	if code != http.StatusCreated {
		t.Fatalf("create: %d %v", code, body)
	}
	if got, _ := category(); got != "Pemuda" {
		t.Fatalf("listed after Pemuda, Pemuda still wins: %s", got)
	}
	rules, _ := st.BipraRules(ctx)
	ids := []string{body["id"].(string)}
	for _, r := range rules {
		if r.Key != "Pemudi" {
			ids = append(ids, `"`+r.ID.Hex()+`"`)
		}
	}
	order := `{"ids":["` + ids[0] + `",` + strings.Join(ids[1:], ",") + `]}`
	if code, body := call(h, admin, http.MethodPut, "/api/admin/bipra/order", order); code != http.StatusOK {
		t.Fatalf("reorder: %d %v", code, body)
	}
	if got, _ := category(); got != "Pemudi" {
		t.Fatalf("after moving it up: %s", got)
	}

	// a flag on top of the main category
	if code, body := call(h, admin, http.MethodPost, "/api/admin/bipra",
		`{"key":"Remaja Akhir","name":"Remaja akhir","kind":"flag","color":"#0b8043","minAge":18,"maxAge":21}`); code != http.StatusCreated {
		t.Fatalf("flag: %d %v", code, body)
	}
	if got, flags := category(); got != "Pemudi" || len(flags) != 1 || flags[0] != "Remaja Akhir" {
		t.Fatalf("with a flag: %s %v", got, flags)
	}
	if code, _ := call(h, admin, http.MethodPost, "/api/admin/bipra", `{"key":"X","name":"x","kind":"main","color":"#000000","minAge":30,"maxAge":10}`); code != http.StatusBadRequest {
		t.Errorf("min > max accepted: %d", code)
	}
}

func TestBipraUserManagesOnlyTheirServices(t *testing.T) {
	_, st, h := testAPI(t)
	ctx := context.Background()
	if err := st.SeedBipra(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.LinkWorshipTypesToBipra(ctx); err != nil {
		t.Fatal(err)
	}
	types, _ := st.ListWorshipTypes(ctx)
	typeID := map[string]string{}
	var pemuda bson.ObjectID
	for _, ty := range types {
		typeID[ty.Name] = ty.ID.Hex()
		if ty.Name == "Pemuda" {
			pemuda = *ty.BipraID
		}
	}
	day := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	kolomService, err := st.CreateService(ctx, model.Service{Kolom: 3, TypeID: types[0].ID, Date: day, Time: "19:00"}) // "Kolom"
	if err != nil {
		t.Fatal(err)
	}
	f, _ := st.CreateFamily(ctx, model.Family{Kolom: 3, Name: "Umboh"})
	young := time.Now().AddDate(-20, 0, -1)
	old := time.Now().AddDate(-45, 0, -1)
	_, _ = st.CreateMember(ctx, model.Member{FamilyID: f.ID, Name: "Budi", Gender: model.GenderMale, MaritalStatus: model.StatusSingle, BirthDate: &young})
	_, _ = st.CreateMember(ctx, model.Member{FamilyID: f.ID, Name: "Erwin", Gender: model.GenderMale, MaritalStatus: "Menikah", BirthDate: &old})

	u := &model.User{ID: bson.NewObjectID(), Username: "pemuda", Role: model.RoleBipra, BipraID: &pemuda}
	service := func(kind string, kolom string) string {
		return `{"kolom":` + kolom + `,"typeId":"` + typeID[kind] + `","date":"2026-11-11T00:00:00Z","time":"18:00"}`
	}
	for _, c := range []struct {
		what, method, path, body string
		want                     int
	}{
		{"a Pemuda service for the whole congregation", "POST", "/api/services", service("Pemuda", "0"), 201},
		{"a Pemuda service in one kolom", "POST", "/api/services", service("Pemuda", "3"), 201},
		{"another category's service", "POST", "/api/services", service("Pria Kaum Bapa (PKB)", "0"), 403},
		{"a kolom service", "POST", "/api/services", service("Kolom", "3"), 403},
		{"turn a kolom service into theirs", "PUT", "/api/services/" + kolomService.ID.Hex(), service("Pemuda", "0"), 403},
		{"delete a kolom service", "DELETE", "/api/services/" + kolomService.ID.Hex(), "", 403},
		{"add a family", "POST", "/api/families", `{"kolom":3,"name":"Baru"}`, 403},
		{"add a member", "POST", "/api/members", `{"familyId":"` + f.ID.Hex() + `","name":"x"}`, 403},
	} {
		if got, body := call(h, u, c.method, c.path, c.body); got != c.want {
			t.Errorf("%s: %d, want %d (%v)", c.what, got, c.want, body)
		}
	}
	// a service without a kolom needs a jenis linked to a category (also for admins)
	admin := &model.User{ID: bson.NewObjectID(), Username: "admin", Role: model.RoleAdmin}
	if got, body := call(h, admin, "POST", "/api/services", service("Kolom", "0")); got != 400 {
		t.Errorf("kolom service without a kolom: %d %v", got, body)
	}

	// private data: their category's members only
	r := auth.WithUser(httptest.NewRequest(http.MethodGet, "/api/members", nil), u)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	var members []model.MemberView
	_ = json.Unmarshal(rec.Body.Bytes(), &members)
	for _, m := range members {
		if (m.BirthDate != nil) != (m.Name == "Budi") {
			t.Errorf("%s (%s): birth date visible = %v", m.Name, m.Category, m.BirthDate != nil)
		}
	}
}
