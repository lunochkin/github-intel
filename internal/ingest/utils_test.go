package ingest

import (
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
