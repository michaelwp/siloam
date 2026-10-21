package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/model"
)

func TestAuditLog(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	if err := s.EnsureAuditIndexes(ctx, 30*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	// a changed retention updates the TTL index instead of failing
	if err := s.EnsureAuditIndexes(ctx, 90*24*time.Hour); err != nil {
		t.Fatalf("changing the retention: %v", err)
	}
	specs, err := s.audit.Indexes().ListSpecifications(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, sp := range specs {
		if sp.Name == auditTTLIndex && (sp.ExpireAfterSeconds == nil || *sp.ExpireAfterSeconds != 90*24*3600) {
			t.Fatalf("TTL = %v", sp.ExpireAfterSeconds)
		}
	}

	day := time.Date(2026, 10, 21, 3, 0, 0, 0, time.UTC)
	add := func(user, category string, status int, at time.Time) {
		if err := s.InsertAudit(ctx, &model.AuditEntry{At: at, Username: user, Category: category, Status: status}); err != nil {
			t.Fatal(err)
		}
	}
	add("admin", "anggota", 200, day)
	add("budi", "masuk", 401, day.Add(time.Hour))
	add("admin", "jadwal", 201, day.Add(2*time.Hour))
	add("admin", "anggota", 204, day.AddDate(0, 0, 1))

	all, err := s.ListAudit(ctx, AuditFilter{})
	if err != nil || len(all) != 4 || all[0].Category != "anggota" || !all[0].At.Equal(day.AddDate(0, 0, 1)) {
		t.Fatalf("newest first: %+v (%v)", all, err)
	}
	failed := true
	for name, c := range map[string]struct {
		f    AuditFilter
		want int
	}{
		"by user":     {AuditFilter{Username: "admin"}, 3},
		"by category": {AuditFilter{Category: "anggota"}, 2},
		"failed only": {AuditFilter{Failed: &failed}, 1},
		"by day":      {AuditFilter{From: day.Truncate(24 * time.Hour), To: day.Truncate(24*time.Hour).AddDate(0, 0, 1)}, 3},
		"next page":   {AuditFilter{Before: all[1].ID, Limit: 1}, 1},
	} {
		got, err := s.ListAudit(ctx, c.f)
		if err != nil || len(got) != c.want {
			t.Errorf("%s: %d entries (%v), want %d", name, len(got), err, c.want)
		}
	}
	names, err := s.AuditUsernames(ctx)
	if err != nil || len(names) != 2 {
		t.Fatalf("usernames = %v (%v)", names, err)
	}
}

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
