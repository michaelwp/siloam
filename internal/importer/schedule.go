package importer

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"siloam/internal/model"
	"siloam/internal/store"
)

// ImportSchedule loads a "Jadwal Ibadah" workbook (first sheet) into the given
// kolom. Columns are found by their headers: Hari / Tanggal, Nama Keluarga
// (a "Kel." cell followed by the name), Pembawa Acara, Pemimpin Ibadah, the
// time (as a day fraction) and Jenis Ibadah (matched to the worship types).
func ImportSchedule(ctx context.Context, s *store.Store, path string, kolom int) (int, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return 0, fmt.Errorf("%s has no sheets", path)
	}
	rows, err := f.GetRows(sheets[0], excelize.Options{RawCellValue: true})
	if err != nil {
		return 0, err
	}
	types, err := s.ListWorshipTypes(ctx)
	if err != nil {
		return 0, err
	}

	var col map[string]int
	var out []model.Service
	for i, row := range rows {
		if col == nil {
			found := headerColumns(row)
			if _, ok := found["date"]; !ok {
				continue // rows above the header (title)
			}
			if missing := missingColumns(found); len(missing) > 0 {
				return 0, fmt.Errorf("row %d: header has no column for %s", i+1, strings.Join(missing, ", "))
			}
			col = found
			continue
		}
		date := serialDate(cell(row, col["date"]))
		if date == nil {
			continue // title, blank or footer row
		}
		host := cell(row, col["host"])
		if strings.EqualFold(strings.TrimSuffix(host, "."), "kel") || host == "" {
			host = cell(row, col["host"]+1)
		}
		jenis := cell(row, col["type"])
		t, ok := matchType(types, jenis)
		if !ok {
			return 0, fmt.Errorf("row %d: unknown jenis ibadah %q", i+1, jenis)
		}
		out = append(out, model.Service{
			Kolom:  kolom,
			TypeID: t.ID,
			Date:   *date,
			Time:   timeCell(row, col),
			Host:   strings.TrimSpace(strings.TrimPrefix(host, "Kel. ")),
			MC:     cell(row, col["mc"]),
			Leader: cell(row, col["leader"]),
		})
	}
	if col == nil {
		return 0, fmt.Errorf("sheet %q: header row (Hari / Tanggal …) not found", sheets[0])
	}
	return len(out), s.InsertServices(ctx, out)
}

// headerColumns returns the column of each field when row is the header row.
func headerColumns(row []string) map[string]int {
	col := map[string]int{}
	for i := range row {
		h := strings.ToLower(cell(row, i))
		switch {
		case strings.Contains(h, "tanggal"):
			col["date"] = i
		case strings.Contains(h, "nama keluarga"):
			col["host"] = i
		case strings.Contains(h, "pembawa"):
			col["mc"] = i
		case strings.Contains(h, "pemimpin"):
			col["leader"] = i
		case h == "time" || h == "jam" || h == "waktu":
			col["time"] = i
		case strings.Contains(h, "jenis"):
			col["type"] = i
		}
	}
	return col
}

// requiredColumns are the headers a schedule must have (the time is
// optional: without it every service is at 19:00).
var requiredColumns = []struct{ key, header string }{
	{"date", "Hari / Tanggal"},
	{"host", "Nama Keluarga"},
	{"mc", "Pembawa Acara"},
	{"leader", "Pemimpin Ibadah"},
	{"type", "Jenis Ibadah"},
}

func missingColumns(col map[string]int) []string {
	var out []string
	for _, c := range requiredColumns {
		if _, ok := col[c.key]; !ok {
			out = append(out, fmt.Sprintf("%q", c.header))
		}
	}
	return out
}

func timeCell(row []string, col map[string]int) string {
	i, ok := col["time"]
	if !ok {
		return "19:00"
	}
	return dayFraction(cell(row, i))
}

// matchType finds a type by name or by the abbreviation in parentheses,
// ignoring case: "kolom" → Kolom, "wki" → Wanita Kaum Ibu (WKI). Empty → the first type.
func matchType(types []model.WorshipType, v string) (model.WorshipType, bool) {
	if len(types) == 0 {
		return model.WorshipType{}, false
	}
	if v == "" {
		return types[0], true
	}
	for _, t := range types {
		abbr := ""
		if i, j := strings.LastIndex(t.Name, "("), strings.LastIndex(t.Name, ")"); i >= 0 && j > i {
			abbr = t.Name[i+1 : j]
		}
		if strings.EqualFold(t.Name, v) || (abbr != "" && strings.EqualFold(abbr, v)) {
			return t, true
		}
	}
	return model.WorshipType{}, false
}

// dayFraction turns Excel's time-of-day (0.7916… = 19:00) into "19:00";
// anything else defaults to 19:00.
func dayFraction(v string) string {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 || f >= 1 {
		return "19:00"
	}
	mins := int(math.Round(f * 24 * 60))
	return fmt.Sprintf("%02d:%02d", mins/60%24, mins%60)
}
