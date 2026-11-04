package api

import (
	"net/http"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/model"
)

// Stats is the congregation's summary that visitors (not signed in) see: only
// counts, never a name or any member's or family's data.
type Stats struct {
	Families        int             `json:"families"`
	Members         int             `json:"members"`
	Male            int             `json:"male"`
	Female          int             `json:"female"`
	Baptized        YesNo           `json:"baptized"`
	Confirmed       YesNo           `json:"confirmed"`
	MaritalStatuses []LabelCount    `json:"maritalStatuses"`
	Categories      []CategoryCount `json:"categories"`
	BirthMonths     [12]int         `json:"birthMonths"` // members born in each month, January first
	FamilySizes     []int           `json:"familySizes"` // [i]: families with i+1 members
}

type YesNo struct {
	Yes     int `json:"yes"`
	No      int `json:"no"`
	Unknown int `json:"unknown"`
}

func (c *YesNo) add(v *bool) {
	switch {
	case v == nil:
		c.Unknown++
	case *v:
		c.Yes++
	default:
		c.No++
	}
}

type LabelCount struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

// CategoryCount is a BIPRA category (its rule, for the age range shown) with
// its members, by gender.
type CategoryCount struct {
	model.BipraCategory
	Count  int `json:"count"`
	Male   int `json:"male"`
	Female int `json:"female"`
}

// maritalStatuses are listed first, in this order; any other the data has follows.
var maritalStatuses = []string{
	model.StatusSingle, "Menikah", "Cerai Hidup / Duda", "Cerai Hidup / Janda", "Cerai Mati / Duda", "Cerai Mati / Janda",
}

func (a *API) stats(w http.ResponseWriter, r *http.Request) {
	fams, err := a.store.ListFamilies(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	members, err := a.store.ListMembers(r.Context(), bson.ObjectID{})
	if err != nil {
		serverError(w, err)
		return
	}
	rules, err := a.store.BipraRules(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, buildStats(fams, members, rules, time.Now()))
}

func buildStats(fams []model.Family, members []model.Member, rules model.BipraRules, now time.Time) Stats {
	byID := map[bson.ObjectID]model.Family{}
	for _, f := range fams {
		byID[f.ID] = f
	}
	st := Stats{Families: len(fams), Members: len(members), FamilySizes: []int{}}
	cats := make([]CategoryCount, len(rules))
	for i, c := range rules {
		cats[i] = CategoryCount{BipraCategory: c}
	}
	statuses := map[string]int{}
	sizes := map[bson.ObjectID]int{}
	for _, m := range members {
		sizes[m.FamilyID]++
		v := model.NewMemberView(m, byID[m.FamilyID], now, rules)
		switch m.Gender {
		case model.GenderMale:
			st.Male++
		case model.GenderFemale:
			st.Female++
		}
		st.Baptized.add(m.Baptized)
		st.Confirmed.add(m.Confirmed)
		if m.MaritalStatus != "" {
			statuses[m.MaritalStatus]++
		}
		if v.BirthMonth != nil {
			st.BirthMonths[*v.BirthMonth-1]++
		}
		for i := range cats {
			c := &cats[i]
			if (c.Kind == model.KindFlag && slices.Contains(v.Flags, c.Key)) || (c.Kind != model.KindFlag && v.Category == c.Key) {
				c.Count++
				switch m.Gender {
				case model.GenderMale:
					c.Male++
				case model.GenderFemale:
					c.Female++
				}
			}
		}
	}
	st.Categories = cats
	for _, s := range maritalStatuses {
		st.MaritalStatuses = append(st.MaritalStatuses, LabelCount{s, statuses[s]})
		delete(statuses, s)
	}
	others := make([]string, 0, len(statuses))
	for s := range statuses {
		others = append(others, s)
	}
	slices.Sort(others)
	for _, s := range others {
		st.MaritalStatuses = append(st.MaritalStatuses, LabelCount{s, statuses[s]})
	}
	for _, f := range fams {
		n := sizes[f.ID]
		if n == 0 {
			continue
		}
		for len(st.FamilySizes) < n {
			st.FamilySizes = append(st.FamilySizes, 0)
		}
		st.FamilySizes[n-1]++
	}
	return st
}
