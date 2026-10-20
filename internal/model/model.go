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
	CreatedAt   time.Time     `bson:"createdAt" json:"createdAt,omitzero"`
	UpdatedAt   time.Time     `bson:"updatedAt" json:"updatedAt,omitzero"`
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
	CreatedAt     time.Time     `bson:"createdAt" json:"createdAt,omitzero"`
	UpdatedAt     time.Time     `bson:"updatedAt" json:"updatedAt,omitzero"`
}

// FamilyView is a Family enriched for API responses.
type FamilyView struct {
	Family
	MemberCount  int          `json:"memberCount"`
	WeddingYears *int         `json:"weddingYears"`
	WeddingMonth *int         `json:"weddingMonth"` // 1–12; also in the public view
	WeddingDay   *int         `json:"weddingDay"`
	Members      []MemberView `json:"members,omitempty"`
}

// MemberView is a Member enriched for API responses.
type MemberView struct {
	Member
	FamilyName string `json:"familyName"`
	Age        *int   `json:"age"`
	AgeText    string `json:"ageText"`
	Category   string `json:"category"`
	Elderly    bool   `json:"elderly"`    // lansia, 60+
	BirthMonth *int   `json:"birthMonth"` // 1–12; also in the public view (birthday lists)
	BirthDay   *int   `json:"birthDay"`
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
		v.BirthMonth, v.BirthDay = monthDay(*m.BirthDate)
	}
	return v
}

func monthDay(t time.Time) (*int, *int) {
	m, d := int(t.Month()), t.Day()
	return &m, &d
}

// Public returns the view for visitors who are not signed in: name, family,
// gender, marital status, baptism/sidi, BIPRA category and birthday (day and
// month). Birth year, age, birth place and family relationship are removed.
func (v MemberView) Public() MemberView {
	return MemberView{
		Member: Member{
			ID:            v.ID,
			FamilyID:      v.FamilyID,
			Order:         v.Order,
			Name:          v.Name,
			Gender:        v.Gender,
			MaritalStatus: v.MaritalStatus,
			Baptized:      v.Baptized,
			Confirmed:     v.Confirmed,
		},
		FamilyName: v.FamilyName,
		Category:   v.Category,
		Elderly:    v.Elderly,
		BirthMonth: v.BirthMonth,
		BirthDay:   v.BirthDay,
	}
}

// Public returns the family view for visitors: name, number, note, member
// count and anniversary (day and month) only. Domicile, the wedding year and
// years married are removed.
func (v FamilyView) Public() FamilyView {
	out := FamilyView{
		Family:       Family{ID: v.ID, No: v.No, Name: v.Name, Note: v.Note},
		MemberCount:  v.MemberCount,
		WeddingMonth: v.WeddingMonth,
		WeddingDay:   v.WeddingDay,
	}
	for _, m := range v.Members {
		out.Members = append(out.Members, m.Public())
	}
	return out
}

func NewFamilyView(f Family, memberCount int, now time.Time) FamilyView {
	v := FamilyView{Family: f, MemberCount: memberCount}
	if f.WeddingDate != nil {
		y, _, _ := Diff(*f.WeddingDate, now)
		v.WeddingYears = &y
		v.WeddingMonth, v.WeddingDay = monthDay(*f.WeddingDate)
	}
	return v
}
