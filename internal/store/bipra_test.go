package store

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/model"
)

func TestBipraCategories(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	// before the server ever started: the defaults, so members have categories
	rules, err := s.BipraRules(ctx)
	if err != nil || len(rules) != 6 {
		t.Fatalf("unseeded: %d %v", len(rules), err)
	}
	if err := s.SeedBipra(ctx); err != nil {
		t.Fatal(err)
	}
	rules, _ = s.BipraRules(ctx)
	if len(rules) != 6 || rules[0].Key != "Anak" || rules[5].Key != model.LansiaKey || rules[0].ID.IsZero() {
		t.Fatalf("seeded: %+v", rules)
	}

	c, err := s.CreateBipra(ctx, model.BipraCategory{Key: "Pemudi", Name: "Pemudi", Kind: model.KindMain, Color: "#8e24aa"})
	if err != nil || c.Order != 7 {
		t.Fatalf("create: %+v %v", c, err)
	}
	if _, err := s.CreateBipra(ctx, model.BipraCategory{Key: "pkb", Name: "x", Kind: model.KindMain, Color: "#000000"}); !errors.Is(err, ErrBipraExists) {
		t.Fatalf("duplicate code (case ignored): %v", err)
	}
	c.Name = "Pemudi Gereja"
	if c, err = s.UpdateBipra(ctx, c); err != nil || c.Order != 7 {
		t.Fatalf("update keeps the order: %+v %v", c, err)
	}

	// move Pemudi to the top
	rules, _ = s.BipraRules(ctx)
	ids := []bson.ObjectID{c.ID}
	for _, r := range rules[:6] {
		ids = append(ids, r.ID)
	}
	if err := s.ReorderBipra(ctx, ids); err != nil {
		t.Fatal(err)
	}
	rules, _ = s.BipraRules(ctx)
	if rules[0].Key != "Pemudi" || rules[1].Key != "Anak" {
		t.Fatalf("reordered: %s, %s", rules[0].Key, rules[1].Key)
	}
	if err := s.ReorderBipra(ctx, ids[:3]); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a partial order is refused: %v", err)
	}

	for _, r := range rules {
		if err := s.DeleteBipra(ctx, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	// all deleted on purpose: no categories, and the defaults do not come back
	if err := s.SeedBipra(ctx); err != nil {
		t.Fatal(err)
	}
	if rules, _ = s.BipraRules(ctx); len(rules) != 0 {
		t.Fatalf("defaults came back: %d", len(rules))
	}
}

func TestBipraLinksAndUsers(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	if err := s.SeedBipra(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedWorshipTypes(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.LinkWorshipTypesToBipra(ctx); err != nil {
		t.Fatal(err)
	}
	rules, _ := s.BipraRules(ctx)
	keyOf := map[bson.ObjectID]string{}
	for _, c := range rules {
		keyOf[c.ID] = c.Key
	}
	types, _ := s.ListWorshipTypes(ctx)
	links := map[string]string{}
	var pemudaType model.WorshipType
	for _, ty := range types {
		if ty.BipraID != nil {
			links[ty.Name] = keyOf[*ty.BipraID]
		}
		if ty.Name == "Pemuda" {
			pemudaType = ty
		}
	}
	want := map[string]string{"Wanita Kaum Ibu (WKI)": "WKI", "Pria Kaum Bapa (PKB)": "PKB", "Pemuda": "Pemuda", "Remaja": "Remaja", "Anak Sekolah Minggu (ASM)": "Anak"}
	if len(links) != len(want) {
		t.Fatalf("links = %v", links)
	}
	for k, v := range want {
		if links[k] != v {
			t.Errorf("%s → %s, want %s", k, links[k], v)
		}
	}

	// a category with a user cannot go; without, it goes and its jenis lose the link
	newUser(t, s, "admin", model.RoleAdmin)
	u := newUser(t, s, "pemuda", model.RoleUser)
	pemuda := *pemudaType.BipraID
	got, err := s.SetRole(ctx, u.ID, model.RoleBipra, 0, &pemuda)
	if err != nil || got.BipraID == nil || *got.BipraID != pemuda || got.Kolom != 0 {
		t.Fatalf("to bipra user: %+v %v", got, err)
	}
	if err := s.DeleteBipra(ctx, pemuda); !errors.Is(err, ErrBipraHasUsers) {
		t.Fatalf("delete an assigned category: %v", err)
	}
	if got, err = s.SetRole(ctx, u.ID, model.RoleUser, 0, nil); err != nil || got.BipraID != nil {
		t.Fatalf("back to regular user keeps the category: %+v %v", got, err)
	}
	if err := s.DeleteBipra(ctx, pemuda); err != nil {
		t.Fatal(err)
	}
	if ty, _ := s.GetWorshipType(ctx, pemudaType.ID); ty.BipraID != nil {
		t.Fatal("the jenis still links to the deleted category")
	}
}
