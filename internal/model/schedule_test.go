package model

import (
	"testing"
	"time"
)

func TestServiceTimes(t *testing.T) {
	wita := time.FixedZone("WITA", 8*3600)
	sv := Service{Date: time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC), Time: "19:00", Host: "Umboh Riedel"}
	v := NewServiceView(sv, WorshipType{Name: "Kolom"}, wita)
	if got := v.Start.UTC().Format(time.RFC3339); got != "2026-01-21T11:00:00Z" {
		t.Errorf("start = %s", got)
	}
	if v.End.Sub(v.Start) != ServiceDuration || v.Title() != "Ibadah Kolom · Kel. Umboh Riedel" {
		t.Errorf("end/title = %v %q", v.End, v.Title())
	}
}
