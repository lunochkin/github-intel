package ingest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// backfillState persists the next hour to ingest (UTC).
type backfillState struct {
	HoursUTCInProgress []string `json:"hours_utc_in_progress"`
	NextHourUTC        string   `json:"next_hour_utc"`
}

func parseNextHourUTC(st backfillState) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, st.NextHourUTC)
	if err != nil {
		t, err = time.Parse(time.RFC3339, st.NextHourUTC)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("state next_hour_utc %q: %w", st.NextHourUTC, err)
	}
	return truncateHourUTC(t), nil
}

var backfillStateMutex sync.Mutex

func addHourUTCInProgress(path string, hour time.Time) error {
	backfillStateMutex.Lock()
	defer backfillStateMutex.Unlock()
	st, loaded, err := loadBackfillState(path)
	if err != nil {
		return err
	} else if !loaded {
		st = backfillState{
			HoursUTCInProgress: []string{},
			NextHourUTC:        "",
		}
	}
	h := hour.UTC().Format(time.RFC3339)
	if !slices.Contains(st.HoursUTCInProgress, h) {
		st.HoursUTCInProgress = append(st.HoursUTCInProgress, h)
	}

	nextHour, err := parseNextHourUTC(st)
	if err != nil && st.NextHourUTC != "" {
		return err
	}
	if err != nil || nextHour.Before(hour) || nextHour.Equal(hour) {
		st.NextHourUTC = hour.Add(time.Hour).UTC().Format(time.RFC3339)
	}
	return saveBackfillState(path, st)
}

func removeHourUTCInProgress(path string, hour time.Time) error {
	backfillStateMutex.Lock()
	defer backfillStateMutex.Unlock()
	st, loaded, err := loadBackfillState(path)
	if err != nil {
		return err
	}
	if !loaded {
		return fmt.Errorf("hour %s not in progress", hour.Format(time.RFC3339))
	}
	key := hour.UTC().Format(time.RFC3339)
	st.HoursUTCInProgress = slices.DeleteFunc(st.HoursUTCInProgress, func(h string) bool {
		return h == key
	})

	doneNext := truncateHourUTC(hour).Add(time.Hour)
	var watermark time.Time
	if st.NextHourUTC != "" {
		cur, err := parseNextHourUTC(st)
		if err != nil {
			watermark = doneNext
		} else if cur.After(doneNext) {
			watermark = cur
		} else {
			watermark = doneNext
		}
	} else {
		watermark = doneNext
	}
	st.NextHourUTC = watermark.UTC().Format(time.RFC3339)

	return saveBackfillState(path, st)
}

// loadBackfillState reads StatePath. If missing, returns (zero, false, nil).
// If the file exists, returns the unmarshaled state and ok=true (NextHourUTC may be empty).
func loadBackfillState(path string) (state backfillState, ok bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return backfillState{}, false, nil
		}
		return backfillState{}, false, err
	}
	var st backfillState
	if err := json.Unmarshal(data, &st); err != nil {
		return backfillState{}, false, fmt.Errorf("state file %s: %w", path, err)
	}
	return st, true, nil
}

func saveBackfillState(path string, st backfillState) error {
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
