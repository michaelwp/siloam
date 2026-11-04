package store

import (
	"context"
	"errors"
	"slices"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"siloam/internal/model"
)

func TestAuditLabelAndAndroidRelease(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	f, err := s.CreateFamily(ctx, model.Family{Kolom: 11, Name: "Umboh"})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.AuditLabel(ctx, "families", f.ID); got != "Kel. Umboh" {
		t.Errorf("family label = %q", got)
	}
	if got := s.AuditLabel(ctx, "families", bson.NewObjectID()); got != "" {
		t.Errorf("missing family label = %q", got)
	}

	if r, err := s.GetAndroidRelease(ctx); err != nil || r != nil {
		t.Fatalf("unset: %v %v", r, err)
	}
	want := model.AndroidRelease{LatestVersion: "1.0.1", DownloadURL: "https://example.org/a.apk", UpdatedBy: "admin"}
	if err := s.SetAndroidRelease(ctx, want); err != nil {
		t.Fatal(err)
	}
	if r, err := s.GetAndroidRelease(ctx); err != nil || r == nil || r.LatestVersion != "1.0.1" || r.UpdatedBy != "admin" {
		t.Fatalf("saved: %+v %v", r, err)
	}
	if err := s.ClearAndroidRelease(ctx); err != nil {
		t.Fatal(err)
	}
	if r, _ := s.GetAndroidRelease(ctx); r != nil {
		t.Fatalf("cleared: %+v", r)
	}
}

func TestKolomUsersRole(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	newUser(t, s, "admin", model.RoleAdmin)
	u := newUser(t, s, "budi", model.RoleUser)
	got, err := s.SetRole(ctx, u.ID, model.RoleScoped, []int{3, 5}, nil)
	if err != nil || got.Role != model.RoleScoped || !slices.Equal(got.Koloms, []int{3, 5}) {
		t.Fatalf("to kolom user: %+v %v", got, err)
	}
	ks, err := s.ListKoloms(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var k3 model.Kolom
	for _, k := range ks {
		if k.Number == 3 {
			k3 = k
		}
	}
	if err := s.DeleteKolom(ctx, k3.ID); !errors.Is(err, ErrKolomHasUsers) {
		t.Fatalf("delete an assigned kolom: %v", err)
	}
	if got, err = s.SetRole(ctx, u.ID, model.RoleUser, nil, nil); err != nil || got.Koloms != nil {
		t.Fatalf("back to regular user keeps the kolom: %+v %v", got, err)
	}
	if err := s.DeleteKolom(ctx, k3.ID); err != nil {
		t.Fatalf("delete once unassigned: %v", err)
	}
}

func TestMigrateUserScopes(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	newUser(t, s, "admin", model.RoleAdmin)
	k := newUser(t, s, "kolom3", model.LegacyRoleKolom)
	b := newUser(t, s, "pemuda", model.LegacyRoleBipra)
	cat := bson.NewObjectID()
	if _, err := s.users.UpdateByID(ctx, k.ID, bson.D{{Key: "$set", Value: bson.D{{Key: "kolom", Value: 3}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.users.UpdateByID(ctx, b.ID, bson.D{{Key: "$set", Value: bson.D{{Key: "bipraId", Value: cat}}}}); err != nil {
		t.Fatal(err)
	}
	for range 2 { // idempotent
		if err := s.MigrateUserScopes(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := s.GetUser(ctx, k.ID); got.Role != model.RoleScoped || !slices.Equal(got.Koloms, []int{3}) || got.BipraIDs != nil {
		t.Errorf("kolom user: %+v", got)
	}
	if got, _ := s.GetUser(ctx, b.ID); got.Role != model.RoleScoped || !slices.Equal(got.BipraIDs, []bson.ObjectID{cat}) || got.Koloms != nil {
		t.Errorf("bipra user: %+v", got)
	}
	if n, _ := s.users.CountDocuments(ctx, bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "kolom", Value: bson.D{{Key: "$exists", Value: true}}}},
		bson.D{{Key: "bipraId", Value: bson.D{{Key: "$exists", Value: true}}}},
	}}}); n != 0 {
		t.Errorf("%d users keep the old fields", n)
	}
}

func TestEnsureCalendarTokenOnce(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	u := newUser(t, s, "budi", model.RoleUser)
	a, err := s.EnsureCalendarToken(ctx, u.ID, "first")
	if err != nil || a.CalendarToken != "first" {
		t.Fatalf("first: %+v %v", a, err)
	}
	b, err := s.EnsureCalendarToken(ctx, u.ID, "second")
	if err != nil || b.CalendarToken != "first" {
		t.Fatalf("second request keeps the first token: %+v %v", b, err)
	}
	c, err := s.SetCalendarToken(ctx, u.ID, "reset")
	if err != nil || c.CalendarToken != "reset" {
		t.Fatalf("reset: %+v %v", c, err)
	}
}

// The sign-in log is kept: an expiring index from an earlier version goes.
func TestLoginLogIsKept(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	if err := s.logins().Indexes().DropAll(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.logins().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "at", Value: -1}}, Options: options.Index().SetExpireAfterSeconds(90 * 24 * 3600),
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.ensureLoginIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	cur, _ := s.logins().Indexes().List(ctx)
	var idx []bson.M
	_ = cur.All(ctx, &idx)
	for _, ix := range idx {
		if _, ttl := ix["expireAfterSeconds"]; ttl {
			t.Errorf("an expiring index is left: %v", ix)
		}
	}
	if len(idx) != 2 { // _id and at
		t.Errorf("indexes: %v", idx)
	}
}
