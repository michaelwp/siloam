package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/model"
)

func dayFiles(t *testing.T, dir string) []string {
	t.Helper()
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range files {
		names = append(names, f.Name())
	}
	return names
}

func TestFileLogWritesDayFilesAndDropsOldOnes(t *testing.T) {
	dir := t.TempDir()
	wita := time.FixedZone("WITA", 8*60*60)
	now := time.Now()
	old := filePrefix + now.AddDate(0, 0, -91).In(wita).Format(time.DateOnly) + fileSuffix
	kept := filePrefix + now.AddDate(0, 0, -89).In(wita).Format(time.DateOnly) + fileSuffix
	for _, name := range []string{old, kept, "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	l, err := OpenFileLog(dir, 90*24*time.Hour, wita)
	if err != nil {
		t.Fatal(err)
	}
	if got := dayFiles(t, dir); slices.Contains(got, old) || !slices.Contains(got, kept) || !slices.Contains(got, "notes.txt") {
		t.Fatalf("after open: %v", got)
	}

	// 23.30 UTC is the next morning in WITA: the day is the app's
	at := time.Date(2026, 10, 24, 23, 30, 0, 0, time.UTC)
	for _, action := range []string{"Tambah anggota", "Hapus anggota"} {
		if err := l.Write(&model.AuditEntry{ID: bson.NewObjectID(), At: at, Action: action}); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "audit-2026-10-25.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var actions []string
	for sc := bufio.NewScanner(f); sc.Scan(); {
		var e model.AuditEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("line %q: %v", sc.Text(), err)
		}
		actions = append(actions, e.Action)
	}
	if !slices.Equal(actions, []string{"Tambah anggota", "Hapus anggota"}) {
		t.Errorf("lines = %v", actions)
	}
}

func TestListFiltersAndPages(t *testing.T) {
	l, err := OpenFileLog(t.TempDir(), 90*24*time.Hour, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	day := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -2).Add(3 * time.Hour)
	add := func(user, category string, status int, at time.Time) {
		if err := l.Write(&model.AuditEntry{ID: bson.NewObjectIDFromTimestamp(at), At: at, Username: user, Category: category, Status: status}); err != nil {
			t.Fatal(err)
		}
	}
	add("admin", "anggota", 200, day)
	add("budi", "masuk", 401, day.Add(time.Hour))
	add("admin", "jadwal", 201, day.Add(2*time.Hour))
	add("admin", "anggota", 204, day.AddDate(0, 0, 1)) // the next day's file

	all, err := l.List(Filter{})
	if err != nil || len(all) != 4 || !all[0].At.Equal(day.AddDate(0, 0, 1)) || !all[3].At.Equal(day) {
		t.Fatalf("newest first: %+v (%v)", all, err)
	}
	failed := true
	for name, c := range map[string]struct {
		f    Filter
		want int
	}{
		"by user":     {Filter{Username: "admin"}, 3},
		"by category": {Filter{Category: "anggota"}, 2},
		"failed only": {Filter{Failed: &failed}, 1},
		"by day":      {Filter{From: day.Truncate(24 * time.Hour), To: day.Truncate(24*time.Hour).AddDate(0, 0, 1)}, 3},
		"next page":   {Filter{Before: all[1].ID, Limit: 1}, 1},
		"limit":       {Filter{Limit: 2}, 2},
	} {
		got, err := l.List(c.f)
		if err != nil || len(got) != c.want {
			t.Errorf("%s: %d entries (%v), want %d", name, len(got), err, c.want)
		}
	}
	if page, _ := l.List(Filter{Before: all[1].ID, Limit: 1}); len(page) != 1 || page[0].ID != all[2].ID {
		t.Errorf("next page = %+v, want %v", page, all[2].ID)
	}
	if names, err := l.Usernames(); err != nil || !slices.Equal(names, []string{"admin", "budi"}) {
		t.Errorf("usernames = %v (%v)", names, err)
	}
}

func TestCloseWritesTheQueuedEntries(t *testing.T) {
	file, err := OpenFileLog(t.TempDir(), 90*24*time.Hour, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	rec := &Recorder{Store: &fakeStore{}, File: file, ClientIP: func(*http.Request) string { return "" }}
	rec.Start()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/koloms", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) })
	h := rec.Middleware(mux)
	for range 3 {
		do(h, http.MethodPost, "/api/koloms", "")
	}
	if err := rec.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got, _ := file.List(Filter{}); len(got) != 3 {
		t.Errorf("written %d entries, want 3", len(got))
	}
	// after Close an entry is dropped, not a panic
	do(h, http.MethodPost, "/api/koloms", "")
}
