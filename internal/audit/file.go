package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/model"
)

const (
	filePrefix = "audit-"
	fileSuffix = ".jsonl"
)

// FileLog appends entries to one JSON-lines file per day
// (audit-2026-10-25.jsonl, by UTC day; older files may be by the app's time
// zone) in a directory, and
// deletes the day files older than the retention when it opens and when the
// day changes.
type FileLog struct {
	dir       string
	retention time.Duration
	loc       *time.Location

	mu  sync.Mutex
	day string // the day of f
	f   *os.File
}

// OpenFileLog creates dir if needed and deletes the files past retention.
func OpenFileLog(dir string, retention time.Duration, loc *time.Location) (*FileLog, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("audit log dir: %w", err)
	}
	l := &FileLog{dir: dir, retention: retention, loc: loc}
	l.prune(time.Now())
	return l, nil
}

// Write appends e to the file of its day.
func (l *FileLog) Write(e *model.AuditEntry) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if day := e.At.In(l.loc).Format(time.DateOnly); day != l.day {
		if l.f != nil {
			_ = l.f.Close()
			l.f, l.day = nil, ""
		}
		f, err := os.OpenFile(filepath.Join(l.dir, filePrefix+day+fileSuffix), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
		if err != nil {
			return err
		}
		l.f, l.day = f, day
		l.prune(e.At)
	}
	_, err = l.f.Write(append(b, '\n'))
	return err
}

// Close closes the current day's file.
func (l *FileLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f, l.day = nil, ""
	return err
}

// Filter narrows List; zero values match everything.
type Filter struct {
	Username string
	Category string
	Failed   *bool         // true: only refused/failed requests; false: only successful ones
	From, To time.Time     // At in [From, To)
	Before   bson.ObjectID // the page after this entry (newest first)
	Limit    int64
}

func (f Filter) match(e *model.AuditEntry) bool {
	switch {
	case !f.Before.IsZero() && e.ID.Hex() >= f.Before.Hex():
	case f.Username != "" && e.Username != f.Username:
	case f.Category != "" && e.Category != f.Category:
	case f.Failed != nil && *f.Failed != (e.Status >= 400):
	case !f.From.IsZero() && e.At.Before(f.From):
	case !f.To.IsZero() && !e.At.Before(f.To):
	default:
		return true
	}
	return false
}

// List returns the entries matching f, newest first (at most f.Limit, 50 by
// default), reading the day files from the newest back.
func (l *FileLog) List(f Filter) ([]model.AuditEntry, error) {
	limit := int(f.Limit)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	days, err := l.days()
	if err != nil {
		return nil, err
	}
	// only the days that can hold a match (an entry's id is made after its time)
	// (a day either side: files written before the days were by UTC were by
	// the app's time zone, which is up to a day off; match() filters exactly)
	last, first := "", ""
	if !f.To.IsZero() {
		last = f.To.Add(-time.Nanosecond).Add(24 * time.Hour).In(l.loc).Format(time.DateOnly)
	}
	if !f.Before.IsZero() {
		if b := f.Before.Timestamp().Add(24 * time.Hour).In(l.loc).Format(time.DateOnly); last == "" || b < last {
			last = b
		}
	}
	if !f.From.IsZero() {
		first = f.From.Add(-24 * time.Hour).In(l.loc).Format(time.DateOnly)
	}
	out := []model.AuditEntry{}
	for i := len(days) - 1; i >= 0 && len(out) < limit; i-- {
		if last != "" && days[i] > last {
			continue
		}
		if days[i] < first {
			break
		}
		entries, err := l.read(days[i])
		if err != nil {
			return nil, err
		}
		for j := range entries {
			if len(out) < limit && f.match(&entries[j]) {
				out = append(out, entries[j])
			}
		}
	}
	return out, nil
}

// Usernames lists the names that appear in the log (for its filter), sorted.
func (l *FileLog) Usernames() ([]string, error) {
	days, err := l.days()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, day := range days {
		entries, err := l.read(day)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.Username != "" {
				seen[e.Username] = true
			}
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	slices.Sort(names)
	return names, nil
}

// read returns a day's entries newest first. A line that does not parse (one
// being written right now) is skipped; a file pruned meanwhile reads as empty.
func (l *FileLog) read(day string) ([]model.AuditEntry, error) {
	f, err := os.Open(filepath.Join(l.dir, filePrefix+day+fileSuffix))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []model.AuditEntry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		var e model.AuditEntry
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			out = append(out, e)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	slices.SortFunc(out, func(a, b model.AuditEntry) int { return strings.Compare(b.ID.Hex(), a.ID.Hex()) })
	return out, nil
}

// days lists the days that have a file, oldest first.
func (l *FileLog) days() ([]string, error) {
	files, err := os.ReadDir(l.dir) // sorted by name, so by day
	if err != nil {
		return nil, err
	}
	var days []string
	for _, f := range files {
		day, ok := strings.CutPrefix(f.Name(), filePrefix)
		day, ok2 := strings.CutSuffix(day, fileSuffix)
		if f.IsDir() || !ok || !ok2 {
			continue
		}
		if _, err := time.Parse(time.DateOnly, day); err == nil {
			days = append(days, day)
		}
	}
	return days, nil
}

// prune deletes the day files from before now minus the retention.
func (l *FileLog) prune(now time.Time) {
	oldest := now.Add(-l.retention).In(l.loc).Format(time.DateOnly)
	days, err := l.days()
	if err != nil {
		log.Printf("audit: prune %s: %v", l.dir, err)
		return
	}
	for _, day := range days {
		if day >= oldest {
			break
		}
		if err := os.Remove(filepath.Join(l.dir, filePrefix+day+fileSuffix)); err != nil {
			log.Printf("audit: prune: %v", err)
		}
	}
}
