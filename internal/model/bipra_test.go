package model

import (
	"slices"
	"testing"
)

// oldCategory is the fixed rule the app used before categories were
// editable; the default rules must give the same answer for everyone.
func oldCategory(age int, gender, status string) (string, bool) {
	isMarried := status != "" && status != StatusSingle
	category := ""
	switch {
	case isMarried || age >= 31:
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

func TestDefaultBipraMatchesTheOldRules(t *testing.T) {
	statuses := []string{"", StatusSingle, "Menikah", "Cerai Hidup / Duda", "Cerai Hidup / Janda", "Cerai Mati / Janda"}
	rules := DefaultBipra()
	// every gender in the data (members without a gender: none today)
	for _, gender := range []string{GenderMale, GenderFemale} {
		for _, status := range statuses {
			for age := 0; age <= 110; age++ {
				want, wantElderly := oldCategory(age, gender, status)
				got, flags := rules.Apply(age, gender, status)
				if got != want || slices.Contains(flags, LansiaKey) != wantElderly {
					t.Fatalf("%d %s %q: %s %v, want %s %v", age, gender, status, got, flags, want, wantElderly)
				}
			}
		}
	}
}

func TestBipraRuleOptions(t *testing.T) {
	rules := BipraRules{
		{Key: "Pemudi", Kind: KindMain, MinAge: intp(17), MaxAge: intp(30), Gender: GenderFemale, Marital: MaritalUnmarried},
		{Key: "Pemuda", Kind: KindMain, MinAge: intp(17), MaxAge: intp(30)}, // after Pemudi: the rest
		{Key: "Baru Menikah", Kind: KindFlag, MaxAge: intp(35), Marital: MaritalMarried},
		{Key: "Lansia", Kind: KindFlag, MinAge: intp(60)},
	}
	for _, c := range []struct {
		age            int
		gender, status string
		want           string
		flags          []string
	}{
		{20, GenderFemale, StatusSingle, "Pemudi", nil},
		{20, GenderMale, StatusSingle, "Pemuda", nil},                     // first match wins
		{25, GenderFemale, "Menikah", "Pemuda", []string{"Baru Menikah"}}, // Pemudi is unmarried only
		{70, GenderMale, "Menikah", "", []string{"Lansia"}},               // no main category matches
	} {
		got, flags := rules.Apply(c.age, c.gender, c.status)
		if got != c.want || !slices.Equal(flags, c.flags) {
			t.Errorf("%d %s %s: %q %v, want %q %v", c.age, c.gender, c.status, got, flags, c.want, c.flags)
		}
	}
}

func TestBipraNormalize(t *testing.T) {
	ok := BipraCategory{Key: " Pemudi ", Name: "  Pemudi   Gereja ", Kind: KindMain, Color: "#e67c73", MinAge: intp(17), MaxAge: intp(30)}
	if err := ok.Normalize(); err != nil || ok.Key != "Pemudi" || ok.Name != "Pemudi Gereja" {
		t.Fatalf("%+v %v", ok, err)
	}
	for name, c := range map[string]BipraCategory{
		"no key":         {Name: "x", Kind: KindMain, Color: "#000000"},
		"no name":        {Key: "X", Kind: KindMain, Color: "#000000"},
		"bad kind":       {Key: "X", Name: "x", Kind: "other", Color: "#000000"},
		"min > max":      {Key: "X", Name: "x", Kind: KindMain, Color: "#000000", MinAge: intp(30), MaxAge: intp(17)},
		"bad gender":     {Key: "X", Name: "x", Kind: KindMain, Color: "#000000", Gender: "L"},
		"married+single": {Key: "X", Name: "x", Kind: KindMain, Color: "#000000", OrMarried: true, Marital: MaritalUnmarried},
		"bad color":      {Key: "X", Name: "x", Kind: KindMain, Color: "red"},
		"negative age":   {Key: "X", Name: "x", Kind: KindMain, Color: "#000000", MinAge: intp(-1)},
	} {
		if err := c.Normalize(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
