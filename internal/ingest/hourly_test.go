package ingest

import (
	"testing"
	"time"
)

func TestClosestFutureArchiveHourUTC(t *testing.T) {
	loc := time.FixedZone("demo", 0)
	// 15:30 UTC -> current hour starts 15:00 (file …-15.json.gz not yet / just landing after 16:00)
	got := ClosestFutureArchiveHourUTC(time.Date(2025, 5, 11, 15, 30, 0, 0, loc))
	want := time.Date(2025, 5, 11, 15, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestHourlyArchiveBaseName(t *testing.T) {
	h := time.Date(2015, 1, 1, 5, 0, 0, 0, time.UTC)
	if g, w := HourlyArchiveBaseName(h), "2015-01-01-05.json.gz"; g != w {
		t.Fatalf("got %q want %q", g, w)
	}
}

func TestPollInterval(t *testing.T) {
	day := time.Date(2025, 5, 11, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		at   time.Time
		want time.Duration
	}{
		{day.Add(14*time.Hour + 0*time.Minute), time.Minute},
		{day.Add(14*time.Hour + 57*time.Minute + 59*time.Second), time.Minute},
		{day.Add(14*time.Hour + 58*time.Minute), 10 * time.Second},
		{day.Add(14*time.Hour + 59*time.Minute + 30*time.Second), 10 * time.Second},
	}
	for _, tc := range tests {
		if g := pollInterval(tc.at); g != tc.want {
			t.Fatalf("at %v got %v want %v", tc.at, g, tc.want)
		}
	}
}
