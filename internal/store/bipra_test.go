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
