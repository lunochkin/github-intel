package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseBackfillInstant(t *testing.T) {
	got, err := ParseBackfillInstant("2011-02-12")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2011, 2, 12, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("date only: got %v want %v", got, want)
	}

	got, err = ParseBackfillInstant("2015-01-01T14:30:00Z")
	if err != nil {
		t.Fatal(err)
	}
	want = time.Date(2015, 1, 1, 14, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("rfc3339: got %v want %v", got, want)
	}
}

func TestBackfillStateRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	next := time.Date(2015, 3, 2, 4, 0, 0, 0, time.UTC)
	if err := saveBackfillState(path, next); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := loadBackfillState(path)
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}
	if !loaded.Equal(next) {
		t.Fatalf("got %v want %v", loaded, next)
	}
}

func TestRunBackfillSkips404ThroughUntil(t *testing.T) {
	if testing.Short() {
		t.Skip("contacts data.gharchive.org")
	}
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	from := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2099, 1, 1, 2, 0, 0, 0, time.UTC)
	opts := BackfillOptions{
		FromUTC:      from,
		UntilUTC:     until,
		StatePath:    statePath,
		PauseBetween: 0,
	}
	if err := RunBackfill(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	next, ok, err := loadBackfillState(statePath)
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}
	wantNext := until.Add(time.Hour)
	if !next.Equal(wantNext) {
		t.Fatalf("next cursor %v want %v", next, wantNext)
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatal(err)
	}
}
