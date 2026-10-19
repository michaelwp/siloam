package model

import (
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
