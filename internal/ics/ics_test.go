package ics

import (
	"strings"
	"testing"
	"time"
)

func TestCalendar(t *testing.T) {
	wita := time.FixedZone("WITA", 8*3600)
	start := time.Date(2026, 1, 20, 19, 0, 0, 0, wita)
	out := string(Calendar("Jadwal Ibadah", []Event{{
		UID: "abc@siloam", Start: start, End: start.Add(90 * time.Minute),
		Summary:     "Ibadah Kolom · Kel. Umboh, Riedel",
		Description: "Khadim: Pdt. Eva; MC: Ibu Frieda\nKolom 11 " + strings.Repeat("panjang ", 12),
	}}))
	for _, want := range []string{
		"BEGIN:VCALENDAR\r\n", "DTSTART:20260120T110000Z\r\n", "DTEND:20260120T123000Z\r\n",
		`SUMMARY:Ibadah Kolom · Kel. Umboh\, Riedel`, `Khadim: Pdt. Eva\; MC: Ibu Frieda\nKolom 11`, "END:VCALENDAR\r\n",
	} {
		if !strings.Contains(strings.ReplaceAll(out, "\r\n ", ""), want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	for _, l := range strings.Split(out, "\r\n") {
		if len(l) > 75 {
			t.Errorf("line longer than 75 octets: %q", l)
		}
	}
}
