// Package importer loads the "Data Base Jemaat" workbook into MongoDB.
//
// Families and members come from the "Data Base Keluarga" sheet; keterangan
// and domisili come from "Daftar Keluarga". The other sheets (HUT Jemaat,
// HUT Pernikahan, Anggota, Kategorial BIPRA) are views over the same data and
// are recomputed by the app, so they are not imported.
package importer

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"siloam/internal/model"
	"siloam/internal/store"
)

const (
	sheetFamilies = "Daftar Keluarga"
	sheetDetail   = "Data Base Keluarga"
	sheetWedding  = "Daftar HUT Pernikahan"
)

type Result struct{ Families, Members int }

type parsedFamily struct {
	family  model.Family
	members []model.Member
}

// Import loads the workbook's families into the given kolom.
func Import(ctx context.Context, s *store.Store, path string, kolom int) (Result, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = f.Close() }()

	families, err := parseDetail(f)
	if err != nil {
		return Result{}, err
	}
	if err := mergeFamilyList(f, families); err != nil {
		return Result{}, err
	}
	if err := mergeWeddingDates(f, families); err != nil {
		return Result{}, err
	}

	var res Result
	for _, pf := range families {
		pf.family.Kolom = kolom
		fam, err := s.CreateFamily(ctx, pf.family)
		if err != nil {
			return res, err
		}
		res.Families++
		if err := s.InsertMembers(ctx, fam.ID, pf.members); err != nil {
			return res, err
		}
		res.Members += len(pf.members)
	}
	return res, nil
}

// parseDetail walks "Data Base Keluarga": a family header row ("Kel." + name),
// an optional "Tanggal Pernikahan ..." row, then numbered member rows.
func parseDetail(f *excelize.File) ([]*parsedFamily, error) {
	rows, err := f.GetRows(sheetDetail, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, fmt.Errorf("sheet %q: %w", sheetDetail, err)
	}
	var out []*parsedFamily
	var cur *parsedFamily
	for _, row := range rows {
		a, b, c, d := cell(row, 0), cell(row, 1), cell(row, 2), cell(row, 3)
		switch {
		case strings.HasPrefix(strings.ToLower(b), "kel") && c != "":
			no, _ := strconv.Atoi(strings.TrimSuffix(a, ".0"))
			cur = &parsedFamily{family: model.Family{No: no, Name: c}}
			out = append(out, cur)
		case cur != nil && strings.HasPrefix(b, "Tanggal Pernikahan"):
			cur.family.WeddingDate = parseIndoDate(b)
		case cur != nil && isNumber(b) && d != "":
			order, _ := strconv.ParseFloat(b, 64)
			cur.members = append(cur.members, model.Member{
				Order:         int(order),
				Name:          d,
				BirthPlace:    cell(row, 4),
				BirthDate:     serialDate(cell(row, 5)),
				Gender:        cell(row, 6),
				MaritalStatus: cell(row, 7),
				Relationship:  cell(row, 8),
				Baptized:      yesNo(cell(row, 9), "Baptis"),
				Confirmed:     yesNo(cell(row, 10), "Sidi"),
			})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("sheet %q: no families found", sheetDetail)
	}
	return out, nil
}

// mergeFamilyList copies keterangan (col D) and domisili (col E) from
// "Daftar Keluarga", matched by family number.
func mergeFamilyList(f *excelize.File, fams []*parsedFamily) error {
	rows, err := f.GetRows(sheetFamilies, excelize.Options{RawCellValue: true})
	if err != nil {
		return fmt.Errorf("sheet %q: %w", sheetFamilies, err)
	}
	byNo := map[int]*parsedFamily{}
	for _, pf := range fams {
		byNo[pf.family.No] = pf
	}
	for _, row := range rows {
		no, err := strconv.ParseFloat(cell(row, 0), 64)
		if err != nil {
			continue
		}
		if pf := byNo[int(no)]; pf != nil {
			pf.family.Note = cell(row, 3)
			pf.family.Domicile = cell(row, 4)
		}
	}
	return nil
}

// mergeWeddingDates fills wedding dates missing from the detail sheet using
// "Daftar HUT Pernikahan", matched by family name.
func mergeWeddingDates(f *excelize.File, fams []*parsedFamily) error {
	if !slices.Contains(f.GetSheetList(), sheetWedding) {
		return nil // optional sheet
	}
	rows, err := f.GetRows(sheetWedding, excelize.Options{RawCellValue: true})
	if err != nil {
		return fmt.Errorf("sheet %q: %w", sheetWedding, err)
	}
	byName := map[string]*time.Time{}
	for _, row := range rows {
		if d := serialDate(cell(row, 3)); d != nil {
			byName[strings.ToLower(cell(row, 2))] = d
		}
	}
	for _, pf := range fams {
		if pf.family.WeddingDate == nil {
			pf.family.WeddingDate = byName[strings.ToLower(pf.family.Name)]
		}
	}
	return nil
}

func cell(row []string, i int) string {
	if i >= len(row) {
		return ""
	}
	return strings.Join(strings.Fields(row[i]), " ")
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// yesNo maps "Baptis"/"Sidi" → true, "Belum" → false, anything else → unknown.
func yesNo(v, yes string) *bool {
	switch {
	case strings.EqualFold(v, yes):
		t := true
		return &t
	case strings.EqualFold(v, "Belum"):
		f := false
		return &f
	}
	return nil
}

func serialDate(v string) *time.Time {
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || n <= 0 {
		return nil
	}
	t, err := excelize.ExcelDateToTime(n, false)
	if err != nil {
		return nil
	}
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return &d
}

var indoMonths = map[string]time.Month{
	"januari": 1, "februari": 2, "maret": 3, "april": 4, "mei": 5, "juni": 6,
	"juli": 7, "agustus": 8, "september": 9, "oktober": 10, "november": 11, "desember": 12,
}

var indoDateRe = regexp.MustCompile(`(\d{1,2})\s+([A-Za-z]+)\s+(\d{4})`)

func parseIndoDate(s string) *time.Time {
	m := indoDateRe.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	month, ok := indoMonths[strings.ToLower(m[2])]
	if !ok {
		return nil
	}
	day, _ := strconv.Atoi(m[1])
	year, _ := strconv.Atoi(m[3])
	d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &d
}
