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
	mux := http.NewServeMux()
	a.Register(mux)
	return a, st, mux
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
