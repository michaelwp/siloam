package importer

import (
	"strings"
	"testing"
)

func TestScheduleHeaders(t *testing.T) {
	full := headerColumns([]string{"", "No.", "Hari / Tanggal", "Nama Keluarga", "", "Pembawa Acara", "Pemimpin Ibadah", "time", "Jenis Ibadah"})
	if m := missingColumns(full); len(m) != 0 || full["date"] != 2 || full["type"] != 8 || full["time"] != 7 {
		t.Fatalf("columns = %v, missing %v", full, m)
	}
	// the leader's column may also be headed "Khadim"
	if c := headerColumns([]string{"No.", "Khadim"}); c["leader"] != 1 {
		t.Fatalf("Khadim header: %v", c)
	}
	partial := headerColumns([]string{"Hari / Tanggal", "Nama Keluarga", "Jenis Ibadah"})
	got := strings.Join(missingColumns(partial), ", ")
	if got != `"Pembawa Acara", "Khadim (Pemimpin Ibadah)"` {
		t.Fatalf("missing = %s", got)
	}
	if timeCell([]string{"x"}, partial) != "19:00" || dayFraction("0.7291666666666666") != "17:30" {
		t.Fatal("time defaults/parsing wrong")
	}
}
