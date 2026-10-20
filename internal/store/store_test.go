package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/model"
)

// These tests need a MongoDB; they use MONGODB_TEST_URI (default local) and a
// throwaway database, and are skipped when none is reachable.
func testStore(t *testing.T) (*Store, string) {
	t.Helper()
	base := os.Getenv("MONGODB_TEST_URI")
	if base == "" {
		base = "mongodb://localhost:27017"
	}
	base = strings.TrimRight(base, "/")
	name := "siloam_test_" + bson.NewObjectID().Hex()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s, err := Open(ctx, base+"/"+name, "")
	if err != nil {
		t.Skipf("no MongoDB at %s: %v", base, err)
	}
	t.Cleanup(func() {
		_ = s.families.Database().Drop(context.Background())
		_ = s.Close(context.Background())
	})
	return s, name
}

func newUser(t *testing.T, s *Store, username, role string) *model.User {
	t.Helper()
	u := &model.User{Username: username, PasswordHash: "old-hash", Role: role, WebAuthnID: []byte(username)}
	if err := s.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	return u
}

const lockFor = 3 * time.Minute

func TestRecordLoginLosesToLock(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	u := newUser(t, s, "staf", model.RoleUser)
	now := time.Now().UTC()
	for range 3 {
		if _, err := s.RecordFailedLogin(ctx, u.ID, 3, now, lockFor); err != nil {
			t.Fatal(err)
		}
	}
	// a correct-password sign-in that was checked before the lock landed
	if _, err := s.RecordLogin(ctx, u.ID, now); !errors.Is(err, ErrLocked) {
		t.Fatalf("RecordLogin on locked account = %v, want ErrLocked", err)
	}
	got, _ := s.GetUser(ctx, u.ID)
	if !got.IsLockedAt(now) || got.FailedLogins != 3 {
		t.Fatalf("lock undone: locked=%v failed=%d", got.IsLockedAt(now), got.FailedLogins)
	}
}

func TestLockExpiresAfterDuration(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	u := newUser(t, s, "staf", model.RoleUser)
	t0 := time.Now().UTC().Truncate(time.Millisecond)
	for range 3 {
		if _, err := s.RecordFailedLogin(ctx, u.ID, 3, t0, lockFor); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := s.GetUser(ctx, u.ID)
	if got.LockedUntil == nil || !got.LockedUntil.Equal(t0.Add(lockFor)) {
		t.Fatalf("lockedUntil = %v, want %v", got.LockedUntil, t0.Add(lockFor))
	}
	if !got.IsLockedAt(t0.Add(lockFor-time.Second)) || got.IsLockedAt(t0.Add(lockFor)) {
		t.Fatal("lock should last exactly lockFor")
	}

	// a failure during the lock (race) neither unlocks nor extends it
	mid := t0.Add(time.Minute)
	got, _ = s.RecordFailedLogin(ctx, u.ID, 3, mid, lockFor)
	if !got.LockedUntil.Equal(t0.Add(lockFor)) {
		t.Fatalf("lock extended to %v", got.LockedUntil)
	}

	// after expiry: a wrong password restarts the count at 1 (no instant relock) …
	after := t0.Add(lockFor + time.Second)
	got, _ = s.RecordFailedLogin(ctx, u.ID, 3, after, lockFor)
	if got.FailedLogins != 1 || got.IsLockedAt(after) {
		t.Fatalf("after expiry: failed=%d locked=%v", got.FailedLogins, got.IsLockedAt(after))
	}
	// … and a correct password signs in and clears everything
	got, err := s.RecordLogin(ctx, u.ID, after)
	if err != nil || got.FailedLogins != 0 || got.LockedUntil != nil || got.LockedAt != nil {
		t.Fatalf("login after expiry: %v failed=%d until=%v", err, got.FailedLogins, got.LockedUntil)
	}
}

func TestLoginDoesNotUndoReset(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	u := newUser(t, s, "staf", model.RoleUser)
	if _, err := s.AddPasskey(ctx, u.ID, model.Passkey{ID: "pk1", Name: "Phone"}); err != nil {
		t.Fatal(err)
	}
	stale, _ := s.GetUser(ctx, u.ID) // loaded by a sign-in in progress
	if _, err := s.ResetPassword(ctx, u.ID, "new-hash", true); err != nil {
		t.Fatal(err)
	}
	// the in-progress passkey sign-in finishes with its stale copy
	if err := s.UsePasskey(ctx, stale.ID, "pk1", []byte("{}")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UsePasskey on removed passkey = %v, want ErrNotFound", err)
	}
	if _, err := s.RecordLogin(ctx, stale.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetUser(ctx, u.ID)
	if got.PasswordHash != "new-hash" || !got.MustChangePassword || len(got.Passkeys) != 0 {
		t.Fatalf("reset undone: hash=%q mustChange=%v passkeys=%d", got.PasswordHash, got.MustChangePassword, len(got.Passkeys))
	}
}

func TestPasskeyEditsKeepPassword(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	u := newUser(t, s, "staf", model.RoleUser)
	if _, err := s.AddPasskey(ctx, u.ID, model.Passkey{ID: "pk1", Name: "Phone"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResetPassword(ctx, u.ID, "new-hash", false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RenamePasskey(ctx, u.ID, "pk1", "Laptop"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetUser(ctx, u.ID)
	if got.PasswordHash != "new-hash" || got.Passkeys[0].Name != "Laptop" {
		t.Fatalf("hash=%q name=%q", got.PasswordHash, got.Passkeys[0].Name)
	}
	if _, err := s.DeletePasskey(ctx, u.ID, "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete unknown passkey = %v", err)
	}
}

func TestLastAdminCannotBeRemovedConcurrently(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	for range 10 {
		a := newUser(t, s, "a"+bson.NewObjectID().Hex()[:6], model.RoleAdmin)
		b := newUser(t, s, "b"+bson.NewObjectID().Hex()[:6], model.RoleAdmin)
		var wg sync.WaitGroup
		wg.Go(func() { _, _ = s.SetRole(ctx, a.ID, model.RoleUser) })
		wg.Go(func() { _ = s.DeleteUser(ctx, b.ID) })
		wg.Wait()
		if n, _ := s.CountAdmins(ctx); n == 0 {
			t.Fatal("no admin left")
		}
		// clean up for the next round
		_, _ = s.users.DeleteMany(ctx, bson.D{})
	}
	only := newUser(t, s, "only", model.RoleAdmin)
	if _, err := s.SetRole(ctx, only.ID, model.RoleUser); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("demote last admin = %v", err)
	}
	if err := s.DeleteUser(ctx, only.ID); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("delete last admin = %v", err)
	}
	if got, err := s.GetUser(ctx, only.ID); err != nil || !got.IsAdmin() {
		t.Fatalf("last admin not restored: %v", err)
	}
}

func TestCopyRefusesSameDatabaseUnderAnotherName(t *testing.T) {
	s, name := testStore(t)
	ctx := context.Background()
	if _, err := s.CreateFamily(ctx, model.Family{Kolom: 1, Name: "Keep"}); err != nil {
		t.Fatal(err)
	}
	// the same server spelled differently
	alias, err := Connect(ctx, "mongodb://127.0.0.1:27017/"+name, "")
	if err != nil {
		t.Skip(err)
	}
	defer func() { _ = alias.Close(ctx) }()
	if _, err := s.CopyFrom(ctx, alias, nil, true); !errors.Is(err, ErrSameDatabase) {
		t.Fatalf("CopyFrom same db = %v, want ErrSameDatabase", err)
	}
	if fams, _ := s.ListFamilies(ctx); len(fams) != 1 {
		t.Fatalf("data lost: %d families", len(fams))
	}
}

func TestNumberingAndCounts(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	f1, _ := s.CreateFamily(ctx, model.Family{Kolom: 1, Name: "A", No: 1})
	_, _ = s.CreateFamily(ctx, model.Family{Kolom: 1, Name: "B", No: 5}) // gap, e.g. from an import
	f3, err := s.CreateFamily(ctx, model.Family{Kolom: 1, Name: "C"})
	if err != nil || f3.No != 6 {
		t.Fatalf("next family no = %d, %v; want 6", f3.No, err)
	}
	for range 2 {
		if _, err := s.CreateMember(ctx, model.Member{FamilyID: f1.ID, Name: "m"}); err != nil {
			t.Fatal(err)
		}
	}
	counts, err := s.MemberCounts(ctx)
	if err != nil || counts[f1.ID] != 2 || counts[f3.ID] != 0 {
		t.Fatalf("counts = %v, %v", counts, err)
	}
}

func TestKolomsSeededAndLegacyFamiliesMigrated(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	ks, err := s.ListKoloms(ctx)
	if err != nil || len(ks) != DefaultKoloms || ks[0].Number != 1 || ks[DefaultKoloms-1].Number != DefaultKoloms {
		t.Fatalf("seeded koloms = %v, %v", ks, err)
	}
	// a family from before koloms existed, and one in a kolom not in the list
	if _, err := s.families.InsertMany(ctx, []any{
		bson.D{{Key: "_id", Value: bson.NewObjectID()}, {Key: "no", Value: 1}, {Key: "name", Value: "Old"}},
		bson.D{{Key: "_id", Value: bson.NewObjectID()}, {Key: "kolom", Value: 40}, {Key: "no", Value: 1}, {Key: "name", Value: "Far"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.MigrateKoloms(ctx); err != nil {
		t.Fatal(err)
	}
	fams, _ := s.ListFamilies(ctx)
	if len(fams) != 2 || fams[0].Kolom != LegacyKolom || fams[1].Kolom != 40 {
		t.Fatalf("families after migration = %+v", fams)
	}
	if ok, _ := s.KolomExists(ctx, 40); !ok {
		t.Fatal("kolom used by a family was not added to the list")
	}
	// a deleted default kolom is not re-seeded
	ks, _ = s.ListKoloms(ctx)
	if err := s.DeleteKolom(ctx, ks[0].ID); err != nil {
		t.Fatal(err)
	}
	_ = s.MigrateKoloms(ctx)
	if ok, _ := s.KolomExists(ctx, 1); ok {
		t.Fatal("deleted kolom came back")
	}
}

func TestFamiliesNumberedPerKolom(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	a1, _ := s.CreateFamily(ctx, model.Family{Kolom: 1, Name: "A1"})
	a2, _ := s.CreateFamily(ctx, model.Family{Kolom: 1, Name: "A2"})
	b1, err := s.CreateFamily(ctx, model.Family{Kolom: 2, Name: "B1"})
	if err != nil || a1.No != 1 || a2.No != 2 || b1.No != 1 {
		t.Fatalf("numbers = %d %d %d, %v", a1.No, a2.No, b1.No, err)
	}
	if _, err := s.CreateFamily(ctx, model.Family{Kolom: 77, Name: "X"}); !errors.Is(err, ErrNoKolom) {
		t.Fatalf("create in unknown kolom = %v, want ErrNoKolom", err)
	}
	// moving A1 to kolom 2 numbers it last there and closes the gap in kolom 1
	a1.Kolom = 2
	moved, err := s.UpdateFamily(ctx, a1)
	if err != nil || moved.No != 2 {
		t.Fatalf("moved no = %d, %v", moved.No, err)
	}
	if got, _ := s.GetFamily(ctx, a2.ID); got.No != 1 {
		t.Fatalf("kolom 1 not renumbered: A2 no = %d", got.No)
	}
	// deleting renumbers only within the kolom
	if err := s.DeleteFamily(ctx, b1.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetFamily(ctx, a1.ID); got.No != 1 {
		t.Fatalf("kolom 2 not renumbered: A1 no = %d", got.No)
	}
	if got, _ := s.GetFamily(ctx, a2.ID); got.No != 1 {
		t.Fatalf("kolom 1 changed by a delete in kolom 2: A2 no = %d", got.No)
	}
}

func TestKolomCreateAndDeleteRules(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	if _, err := s.CreateKolom(ctx, 3); !errors.Is(err, ErrKolomExists) {
		t.Fatalf("duplicate kolom = %v, want ErrKolomExists", err)
	}
	k, err := s.CreateKolom(ctx, 13)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateFamily(ctx, model.Family{Kolom: 13, Name: "F"}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteKolom(ctx, k.ID); !errors.Is(err, ErrKolomInUse) {
		t.Fatalf("delete used kolom = %v, want ErrKolomInUse", err)
	}
}

func TestCeremonyIsSingleUseAndExpires(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	now := time.Now()
	if err := s.PutCeremony(ctx, "c1", []byte("state"), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if b, err := s.TakeCeremony(ctx, "c1", now); err != nil || string(b) != "state" {
		t.Fatalf("take = %q, %v", b, err)
	}
	if _, err := s.TakeCeremony(ctx, "c1", now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second take = %v, want ErrNotFound", err)
	}
	_ = s.PutCeremony(ctx, "c2", []byte("x"), now.Add(time.Minute))
	if _, err := s.TakeCeremony(ctx, "c2", now.Add(2*time.Minute)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired take = %v, want ErrNotFound", err)
	}
}
