package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/auth"
	"siloam/internal/model"
)

// callJSON sends a request as u and decodes the body into out.
func callJSON(h http.Handler, u *model.User, method, path string, out any) int {
	r := auth.WithUser(httptest.NewRequest(method, path, strings.NewReader("")), u)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	_ = json.Unmarshal(rec.Body.Bytes(), out)
	return rec.Code
}

func TestKas(t *testing.T) {
	_, st, h := testAPI(t)
	ctx := context.Background()
	for _, k := range []int{3, 5} {
		if err := st.CreateKolomIfMissing(ctx, k); err != nil {
			t.Fatal(err)
		}
	}
	types, _ := st.ListWorshipTypes(ctx)
	svc3, _ := st.CreateService(ctx, model.Service{Kolom: 3, TypeID: types[0].ID, Date: testToday.AddDate(0, 0, -2), Time: "19:00", Host: "Umboh"})
	svc5, _ := st.CreateService(ctx, model.Service{Kolom: 5, TypeID: types[0].ID, Date: testToday.AddDate(0, 0, -2), Time: "19:00"})

	admin := &model.User{ID: bson.NewObjectID(), Username: "admin", Role: model.RoleAdmin}
	treasurer := &model.User{ID: bson.NewObjectID(), Username: "bendahara", Role: model.RoleTreasurer, Koloms: []int{3}}
	user := &model.User{ID: bson.NewObjectID(), Username: "pengguna", Role: model.RoleUser}
	kolom5 := &model.User{ID: bson.NewObjectID(), Username: "k5", Role: model.RoleScoped, Koloms: []int{5}}

	in := func(kolom int, kind, amount, service, desc string) string {
		return `{"kolom":` + strconv.Itoa(kolom) + `,"kind":"` + kind + `","date":"2026-10-20T00:00:00Z","amount":` + amount +
			`,"serviceId":"` + service + `","description":"` + desc + `"}`
	}
	for _, c := range []struct {
		name string
		u    *model.User
		body string
		want int
	}{
		{"treasurer: offering of an ibadah", treasurer, in(3, "in", "1250000", svc3.ID.Hex(), ""), http.StatusCreated},
		{"treasurer: other source", treasurer, in(3, "in", "500000", "", "Sumbangan  Kel. Lumi"), http.StatusCreated},
		{"treasurer: cash out", treasurer, in(3, "out", "300000", "", "Konsumsi ibadah"), http.StatusCreated},
		{"treasurer: no description", treasurer, in(3, "out", "300000", "", ""), http.StatusBadRequest},
		{"treasurer: another kolom", treasurer, in(5, "in", "1000", "", "x"), http.StatusForbidden},
		{"treasurer: another kolom's ibadah", treasurer, in(3, "in", "1000", svc5.ID.Hex(), ""), http.StatusBadRequest},
		{"zero", treasurer, in(3, "in", "0", "", "x"), http.StatusBadRequest},
		{"bad kind", treasurer, in(3, "gift", "10", "", "x"), http.StatusBadRequest},
		{"regular user", user, in(3, "in", "1000", "", "x"), http.StatusForbidden},
		{"kolom manager", kolom5, in(5, "in", "1000", "", "x"), http.StatusForbidden},
		{"admin: any kolom", admin, in(5, "in", "2000", svc5.ID.Hex(), ""), http.StatusCreated},
	} {
		if got, out := call(h, c.u, "POST", "/api/kas", c.body); got != c.want {
			t.Errorf("%s: %d, want %d (%v)", c.name, got, c.want, out)
		}
	}

	code, book := call(h, treasurer, "GET", "/api/kas?kolom=3", "")
	if code != http.StatusOK || book["balance"] != float64(1450000) || book["canEdit"] != true {
		t.Fatalf("kolom 3 book: %d %v", code, book)
	}
	entries := book["entries"].([]any)
	first := entries[0].(map[string]any)
	if first["serviceLabel"] != "Ibadah Kolom · Kel. Umboh · 2026-10-20" || first["createdBy"] != "bendahara" {
		t.Errorf("first entry: %v", first)
	}
	if entries[1].(map[string]any)["description"] != "Sumbangan Kel. Lumi" {
		t.Errorf("description not cleaned: %v", entries[1])
	}
	if code, _ := call(h, treasurer, "GET", "/api/kas?kolom=5", ""); code != http.StatusForbidden {
		t.Errorf("treasurer reads another kolom: %d", code)
	}
	if code, book := call(h, kolom5, "GET", "/api/kas?kolom=5", ""); code != http.StatusOK || book["canEdit"] != false {
		t.Errorf("kolom manager reads their kolom: %d %v", code, book)
	}
	if code, book := call(h, user, "GET", "/api/kas?kolom=3", ""); code != http.StatusOK || book["canEdit"] != false {
		t.Errorf("regular user reads: %d %v", code, book)
	}

	// change, move away (refused) and delete
	id := first["id"].(string)
	if code, out := call(h, treasurer, "PUT", "/api/kas/"+id, in(3, "in", "1300000", svc3.ID.Hex(), "Persembahan")); code != http.StatusOK || out["updatedBy"] != "bendahara" {
		t.Errorf("update: %d %v", code, out)
	}
	if code, _ := call(h, treasurer, "PUT", "/api/kas/"+id, in(5, "in", "1300000", "", "x")); code != http.StatusForbidden {
		t.Errorf("move to another kolom: %d", code)
	}
	// its ibadah deleted: the entry keeps the link and the name
	if err := st.DeleteService(ctx, svc3.ID); err != nil {
		t.Fatal(err)
	}
	if code, out := call(h, treasurer, "PUT", "/api/kas/"+id, in(3, "in", "1400000", svc3.ID.Hex(), "")); code != http.StatusOK || out["serviceLabel"] != "Ibadah Kolom · Kel. Umboh · 2026-10-20" {
		t.Errorf("update after the ibadah was deleted: %d %v", code, out)
	}
	if code, _ := call(h, kolom5, "DELETE", "/api/kas/"+id, ""); code != http.StatusForbidden {
		t.Errorf("delete by a non-treasurer: %d", code)
	}
	if code, _ := call(h, treasurer, "DELETE", "/api/kas/"+id, ""); code != http.StatusNoContent {
		t.Errorf("delete: %d", code)
	}

	sums, err := st.KasSummaries(ctx, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || len(sums) != 2 || sums[0].In != 500000 || sums[0].Out != 300000 || sums[1].In != 2000 ||
		sums[0].MonthIn != 500000 || sums[0].MonthOut != 300000 {
		t.Errorf("summaries: %v %+v", err, sums)
	}
	if sums, _ := st.KasSummaries(ctx, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)); sums[0].MonthIn != 0 || sums[0].In != 500000 {
		t.Errorf("another month: %+v", sums)
	}
	// the treasurer's summary: only their kolom, this month by default (testToday: October 2026)
	var list []model.KasSummary
	if code := callJSON(h, treasurer, "GET", "/api/kas/summary", &list); code != http.StatusOK || len(list) != 1 || list[0].Kolom != 3 || list[0].Month != "2026-10" || list[0].MonthIn != 500000 {
		t.Errorf("summary: %d %+v", code, list)
	}
	if code := callJSON(h, treasurer, "GET", "/api/kas/summary?month=2026-11", &list); code != http.StatusOK || list[0].MonthIn != 0 {
		t.Errorf("summary of November: %d %+v", code, list)
	}
	if code := callJSON(h, treasurer, "GET", "/api/kas/summary?month=nov", &list); code != http.StatusBadRequest {
		t.Errorf("bad month: %d", code)
	}
	// a kolom with kas cannot be deleted
	ks, _ := st.ListKoloms(ctx)
	for _, k := range ks {
		if k.Number == 5 {
			if err := st.DeleteKolom(ctx, k.ID); err == nil {
				t.Error("deleted a kolom with kas")
			}
		}
	}
}
