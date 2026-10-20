package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestDiff(t *testing.T) {
	cases := []struct {
		from, to time.Time
		y, m, d  int
	}{
		{date(2025, 1, 31), date(2026, 10, 20), 1, 8, 19},
		{date(2000, 3, 15), date(2026, 3, 15), 26, 0, 0},
		{date(2000, 3, 16), date(2026, 3, 15), 25, 11, 27},
		{date(2027, 1, 1), date(2026, 1, 1), 0, 0, 0},
	}
	for _, c := range cases {
		y, m, d := Diff(c.from, c.to)
		if y != c.y || m != c.m || d != c.d {
			t.Errorf("Diff(%v,%v) = %d/%d/%d, want %d/%d/%d", c.from, c.to, y, m, d, c.y, c.m, c.d)
		}
	}
}

func TestCategory(t *testing.T) {
	cases := []struct {
		age            int
		gender, status string
		want           string
		elderly        bool
	}{
		{0, GenderFemale, StatusSingle, "Anak", false},
		{11, GenderMale, StatusSingle, "Anak", false},
		{12, GenderMale, StatusSingle, "Remaja", false},
		{16, GenderFemale, StatusSingle, "Remaja", false},
		{17, GenderMale, StatusSingle, "Pemuda", false},
		{30, GenderFemale, StatusSingle, "Pemuda", false},
		{31, GenderMale, StatusSingle, "PKB", false},
		{28, GenderFemale, "Menikah", "WKI", false},
		{67, GenderFemale, StatusSingle, "WKI", true},
		{72, GenderMale, "Menikah", "PKB", true},
	}
	for _, c := range cases {
		got, elderly := Category(c.age, c.gender, c.status)
		if got != c.want || elderly != c.elderly {
			t.Errorf("Category(%d,%s,%s) = %s/%v, want %s/%v", c.age, c.gender, c.status, got, elderly, c.want, c.elderly)
		}
	}
}

func TestPublicViewsHidePrivateData(t *testing.T) {
	now := date(2026, 10, 20)
	yes := true
	m := Member{
		Name: "Danny Najoan", Gender: GenderMale, BirthPlace: "Gorontalo",
		BirthDate: ptr(date(1992, 11, 13)), MaritalStatus: "Menikah",
		Relationship: "Kepala Keluarga", Baptized: &yes, Confirmed: &yes,
	}
	pub := NewMemberView(m, Family{Name: "Najoan Ismawi", Kolom: 11}, now).Public()
	b, _ := json.Marshal(pub)
	for _, private := range []string{"Gorontalo", "1992", "Kepala Keluarga", "Tahun", `"age":3`} {
		if strings.Contains(string(b), private) {
			t.Errorf("public member JSON leaks %q: %s", private, b)
		}
	}
	if pub.Name != m.Name || pub.FamilyName != "Najoan Ismawi" || pub.Kolom != 11 || pub.Category != "PKB" ||
		*pub.BirthMonth != 11 || *pub.BirthDay != 13 ||
		pub.MaritalStatus != "Menikah" || pub.Baptized == nil || !*pub.Baptized || pub.Confirmed == nil || !*pub.Confirmed {
		t.Errorf("public member lost public fields: %+v", pub)
	}

	f := Family{Kolom: 11, Name: "Najoan Ismawi", Domicile: "Kairagi", WeddingDate: ptr(date(2021, 9, 10))}
	fv := NewFamilyView(f, 3, now)
	fv.Members = []MemberView{NewMemberView(m, f, now)}
	fb, _ := json.Marshal(fv.Public())
	for _, private := range []string{"Kairagi", "2021", `"weddingYears":5`, "Gorontalo"} {
		if strings.Contains(string(fb), private) {
			t.Errorf("public family JSON leaks %q: %s", private, fb)
		}
	}
	if !strings.Contains(string(fb), `"weddingMonth":9`) || !strings.Contains(string(fb), `"memberCount":3`) ||
		!strings.Contains(string(fb), `"kolom":11`) {
		t.Errorf("public family lost public fields: %s", fb)
	}
}

func ptr[T any](v T) *T { return &v }
