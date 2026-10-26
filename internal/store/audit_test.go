package store

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

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
	got, err := s.SetRole(ctx, u.ID, model.RoleKolom, 3, nil)
	if err != nil || got.Role != model.RoleKolom || got.Kolom != 3 {
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
	if got, err = s.SetRole(ctx, u.ID, model.RoleUser, 0, nil); err != nil || got.Kolom != 0 {
		t.Fatalf("back to regular user keeps the kolom: %+v %v", got, err)
	}
	if err := s.DeleteKolom(ctx, k3.ID); err != nil {
		t.Fatalf("delete once unassigned: %v", err)
	}
}
