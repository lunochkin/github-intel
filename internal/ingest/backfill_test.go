package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackfillStateRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	next := time.Date(2015, 3, 2, 4, 0, 0, 0, time.UTC)
	in := backfillState{NextHourUTC: next.UTC().Format(time.RFC3339)}
	if err := saveBackfillState(path, in); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := loadBackfillState(path)
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}
	parsed, err := parseNextHourUTC(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.Equal(next) {
		t.Fatalf("got %v want %v", parsed, next)
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
		Workers:      2,
	}
	if err := RunBackfill(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	st, ok, err := loadBackfillState(statePath)
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}
	nextHour, err := parseNextHourUTC(st)
	if err != nil {
		t.Fatal(err)
	}
	wantNext := until.Add(time.Hour)
	if !nextHour.Equal(wantNext) {
		t.Fatalf("next cursor %v want %v", nextHour, wantNext)
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatal(err)
	}
}
