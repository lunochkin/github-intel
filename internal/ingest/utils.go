package ingest

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func sleepOrDone(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// ParseBackfillInstant parses RFC3339 or UTC date "2006-01-02" and returns that instant truncated to the hour.
func ParseBackfillInstant(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return truncateHourUTC(t), nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: use RFC3339 or YYYY-MM-DD (UTC)", s)
	}
	return truncateHourUTC(t), nil
}

func truncateHourUTC(t time.Time) time.Time {
	return t.UTC().Truncate(time.Hour)
}
