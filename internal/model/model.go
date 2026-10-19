// Package model holds the church-member domain types and the derived values
// (age, BIPRA category, anniversary years) that the spreadsheet used formulas for.
package model

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	GenderMale   = "Laki Laki"
	GenderFemale = "Perempuan"

	StatusSingle = "Belum Menikah"
)

// Family is one row of "Daftar Keluarga" plus the wedding date from
// "Data Base Keluarga" / "Daftar HUT Pernikahan".
type Family struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	No          int           `bson:"no" json:"no"`
	Name        string        `bson:"name" json:"name"`
	Note        string        `bson:"note" json:"note"` // keterangan, e.g. "Simpatisan"
	Domicile    string        `bson:"domicile" json:"domicile"`
	WeddingDate *time.Time    `bson:"weddingDate" json:"weddingDate"`
	CreatedAt   time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time     `bson:"updatedAt" json:"updatedAt"`
}

// Member is one person in a family ("Data Base Keluarga" detail rows).
type Member struct {
	ID            bson.ObjectID `bson:"_id,omitempty" json:"id"`
	FamilyID      bson.ObjectID `bson:"familyId" json:"familyId"`
	Order         int           `bson:"order" json:"order"`
	Name          string        `bson:"name" json:"name"`
	BirthPlace    string        `bson:"birthPlace" json:"birthPlace"`
	BirthDate     *time.Time    `bson:"birthDate" json:"birthDate"`
	Gender        string        `bson:"gender" json:"gender"`
	MaritalStatus string        `bson:"maritalStatus" json:"maritalStatus"`
	Relationship  string        `bson:"relationship" json:"relationship"` // hubungan keluarga
	Baptized      *bool         `bson:"baptized" json:"baptized"`
	Confirmed     *bool         `bson:"confirmed" json:"confirmed"` // sidi
	CreatedAt     time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time     `bson:"updatedAt" json:"updatedAt"`
}

// FamilyView is a Family enriched for API responses.
type FamilyView struct {
	Family
	MemberCount  int      `json:"memberCount"`
	WeddingYears *int     `json:"weddingYears"`
	Members      []Member `json:"members,omitempty"`
}

// MemberView is a Member enriched for API responses.
type MemberView struct {
	Member
	FamilyName string `json:"familyName"`
	Age        *int   `json:"age"`
	AgeText    string `json:"ageText"`
	Category   string `json:"category"`
	Elderly    bool   `json:"elderly"` // lansia, 60+
}

// Diff returns the whole years, months and days between from and to,
// matching Excel's DATEDIF "Y", "YM" and "MD".
func Diff(from, to time.Time) (y, m, d int) {
	from = dateOnly(from)
	to = dateOnly(to)
	if to.Before(from) {
		return 0, 0, 0
	}
	y = to.Year() - from.Year()
	m = int(to.Month()) - int(from.Month())
	d = to.Day() - from.Day()
	if d < 0 {
		m--
		// days in the month preceding `to`
		d += time.Date(to.Year(), to.Month(), 0, 0, 0, 0, 0, time.UTC).Day()
	}
	if m < 0 {
		y--
		m += 12
	}
	return y, m, d
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Category applies the BIPRA rules from the "Kategorial BIPRA" sheet:
// Anak (ASM) 0–11, Remaja 12–16, Pemuda 17–30, PKB/WKI 31+ (or married),
// and Lansia 60+ as an additional flag.
func Category(age int, gender, maritalStatus string) (category string, elderly bool) {
	married := maritalStatus != "" && maritalStatus != StatusSingle
	switch {
	case married || age >= 31:
		if gender == GenderFemale {
			category = "WKI"
		} else {
			category = "PKB"
		}
	case age <= 11:
		category = "Anak"
	case age <= 16:
		category = "Remaja"
	default:
		category = "Pemuda"
	}
	return category, age >= 60
}

func NewMemberView(m Member, familyName string, now time.Time) MemberView {
	v := MemberView{Member: m, FamilyName: familyName}
	if m.BirthDate != nil {
		y, mo, d := Diff(*m.BirthDate, now)
		v.Age = &y
		v.AgeText = fmt.Sprintf("%d Tahun %d Bulan %d Hari", y, mo, d)
		v.Category, v.Elderly = Category(y, m.Gender, m.MaritalStatus)
	}
	return v
}

func NewFamilyView(f Family, memberCount int, now time.Time) FamilyView {
	v := FamilyView{Family: f, MemberCount: memberCount}
	if f.WeddingDate != nil {
		y, _, _ := Diff(*f.WeddingDate, now)
		v.WeddingYears = &y
	}
	return v
}
