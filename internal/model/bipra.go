package model

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Kinds of BIPRA category.
const (
	// KindMain: every member gets at most one, the first (by Order) whose rule matches.
	KindMain = "main"
	// KindFlag: an extra mark on top of the main category (e.g. Lansia), for every member it matches.
	KindFlag = "flag"
)

// Marital conditions of a rule.
const (
	MaritalAny       = ""
	MaritalMarried   = "married"   // any status other than "Belum Menikah" (incl. widowed, divorced)
	MaritalUnmarried = "unmarried" // "Belum Menikah" or not filled in
)

// LansiaKey is the default flag for the elderly; MemberView.Elderly mirrors it
// for clients that predate flags (the Android app 1.0.0).
const LansiaKey = "Lansia"

// BipraCategory is a "kategorial BIPRA" group and the rule that puts members
// in it, managed by admins. All conditions must hold; a member without a
// birth date (no age) is in no category.
type BipraCategory struct {
	ID    bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Key   string        `bson:"key" json:"key"`     // short code, e.g. "PKB" (what members carry)
	Name  string        `bson:"name" json:"name"`   // e.g. "Anak (ASM)"
	Kind  string        `bson:"kind" json:"kind"`   // KindMain or KindFlag
	Order int           `bson:"order" json:"order"` // list order, and priority among main categories
	Color string        `bson:"color" json:"color"` // "#3f51b5"

	MinAge    *int   `bson:"minAge,omitempty" json:"minAge"` // inclusive; nil = no lower bound
	MaxAge    *int   `bson:"maxAge,omitempty" json:"maxAge"` // inclusive; nil = no upper bound
	Gender    string `bson:"gender" json:"gender"`           // "" any, or GenderMale / GenderFemale
	Marital   string `bson:"marital" json:"marital"`         // MaritalAny / MaritalMarried / MaritalUnmarried
	OrMarried bool   `bson:"orMarried" json:"orMarried"`     // married members pass the age condition whatever their age
}

func married(status string) bool { return status != "" && status != StatusSingle }

// Matches reports whether a member of this age, gender and marital status
// falls under the rule.
func (c BipraCategory) Matches(age int, gender, status string) bool {
	ageOK := (c.MinAge == nil || age >= *c.MinAge) && (c.MaxAge == nil || age <= *c.MaxAge)
	if c.OrMarried && married(status) {
		ageOK = true // married members pass the age condition whatever their age
	}
	if !ageOK {
		return false
	}
	if c.Gender != "" && c.Gender != gender {
		return false
	}
	switch c.Marital {
	case MaritalMarried:
		return married(status)
	case MaritalUnmarried:
		return !married(status)
	}
	return true
}

var keyRe = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} ()/.-]{0,19}$`)

// Normalize trims the fields and checks them.
func (c *BipraCategory) Normalize() error {
	c.Key = strings.Join(strings.Fields(c.Key), " ")
	c.Name = strings.Join(strings.Fields(c.Name), " ")
	switch {
	case !keyRe.MatchString(c.Key):
		return errors.New("kode wajib diisi: huruf/angka, paling banyak 20 karakter")
	case c.Name == "":
		return errors.New("nama kategori wajib diisi")
	case c.Kind != KindMain && c.Kind != KindFlag:
		return errors.New("jenis kategori tidak valid")
	case c.Gender != "" && c.Gender != GenderMale && c.Gender != GenderFemale:
		return errors.New("jenis kelamin tidak valid")
	case c.Marital != MaritalAny && c.Marital != MaritalMarried && c.Marital != MaritalUnmarried:
		return errors.New("status pernikahan tidak valid")
	}
	for _, a := range []*int{c.MinAge, c.MaxAge} {
		if a != nil && (*a < 0 || *a > 150) {
			return errors.New("umur harus antara 0 dan 150")
		}
	}
	if c.MinAge != nil && c.MaxAge != nil && *c.MinAge > *c.MaxAge {
		return errors.New("umur minimal lebih besar dari umur maksimal")
	}
	if c.OrMarried && c.Marital == MaritalUnmarried {
		return errors.New(`"juga yang sudah menikah" tidak bisa digabung dengan status belum menikah`)
	}
	if !regexp.MustCompile(`^#[0-9a-fA-F]{6}$`).MatchString(c.Color) {
		return fmt.Errorf("warna tidak valid: %q", c.Color)
	}
	return nil
}

// BipraRules are the categories in Order.
type BipraRules []BipraCategory

// Apply returns the member's main category key ("" if none matches) and the
// keys of the flags that match.
func (rules BipraRules) Apply(age int, gender, status string) (category string, flags []string) {
	for _, c := range rules {
		if !c.Matches(age, gender, status) {
			continue
		}
		if c.Kind == KindFlag {
			flags = append(flags, c.Key)
		} else if category == "" {
			category = c.Key
		}
	}
	return category, flags
}

// Sorted returns the rules in Order (stable for equal orders).
func (rules BipraRules) Sorted() BipraRules {
	out := slices.Clone(rules)
	slices.SortStableFunc(out, func(a, b BipraCategory) int { return a.Order - b.Order })
	return out
}

func intp(n int) *int { return &n }

// DefaultBipra are the categories of the "Kategorial BIPRA" sheet: Anak (ASM)
// 0–11, Remaja 12–16, Pemuda 17–30 (unmarried), PKB/WKI 31+ or married, and
// Lansia 60+ as a flag. A new database starts with them.
func DefaultBipra() BipraRules {
	return BipraRules{
		{Key: "Anak", Name: "Anak (ASM)", Kind: KindMain, Order: 1, Color: "#e67c73", MinAge: intp(0), MaxAge: intp(11), Marital: MaritalUnmarried},
		{Key: "Remaja", Name: "Remaja", Kind: KindMain, Order: 2, Color: "#8e24aa", MinAge: intp(12), MaxAge: intp(16), Marital: MaritalUnmarried},
		{Key: "Pemuda", Name: "Pemuda", Kind: KindMain, Order: 3, Color: "#0b8043", MinAge: intp(17), MaxAge: intp(30), Marital: MaritalUnmarried},
		{Key: "PKB", Name: "PKB", Kind: KindMain, Order: 4, Color: "#3f51b5", MinAge: intp(31), Gender: GenderMale, OrMarried: true},
		{Key: "WKI", Name: "WKI", Kind: KindMain, Order: 5, Color: "#f4511e", MinAge: intp(31), Gender: GenderFemale, OrMarried: true},
		{Key: LansiaKey, Name: "Lansia", Kind: KindFlag, Order: 6, Color: "#616161", MinAge: intp(60)},
	}
}
