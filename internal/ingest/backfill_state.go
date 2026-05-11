package ingest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// backfillState persists the next hour to ingest (UTC).
type backfillState struct {
	NextHourUTC string `json:"next_hour_utc"`
}

// loadBackfillState reads StatePath. If missing or invalid, returns (zero, false, nil).
func loadBackfillState(path string) (nextHour time.Time, ok bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	var st backfillState
	if err := json.Unmarshal(data, &st); err != nil {
		return time.Time{}, false, fmt.Errorf("state file %s: %w", path, err)
	}
	if st.NextHourUTC == "" {
		return time.Time{}, false, nil
	}
	t, err := time.Parse(time.RFC3339Nano, st.NextHourUTC)
	if err != nil {
		t, err = time.Parse(time.RFC3339, st.NextHourUTC)
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("state next_hour_utc %q: %w", st.NextHourUTC, err)
	}
	return truncateHourUTC(t), true, nil
}

func saveBackfillState(path string, nextHour time.Time) error {
	nextHour = truncateHourUTC(nextHour)
	st := backfillState{NextHourUTC: nextHour.UTC().Format(time.RFC3339)}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	committed = true
	return nil
}
