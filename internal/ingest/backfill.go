package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultBackfillStartUTC is the first date GH Archive publishes hourly files (UTC).
var DefaultBackfillStartUTC = time.Date(2011, 2, 12, 0, 0, 0, 0, time.UTC)

// BackfillOptions configures a sequential historical ingest run.
type BackfillOptions struct {
	// FromUTC is the first archive hour to consider (truncated to hour). If Before DefaultBackfillStartUTC, DefaultBackfillStartUTC is used.
	FromUTC time.Time
	// UntilUTC is the last archive hour to include (inclusive), truncated to hour.
	UntilUTC time.Time
	// StatePath is a JSON file storing the next hour to process after the last success or skip.
	StatePath string
	// PauseBetween is slept after each successful or skipped hour ( politeness ).
	PauseBetween time.Duration
	// MaxIngestAttempts is retries per hour when ingest fails (e.g. 5xx during download).
	MaxIngestAttempts int
}

// backfillState persists the next hour to ingest (UTC).
type backfillState struct {
	NextHourUTC string `json:"next_hour_utc"`
}

func truncateHourUTC(t time.Time) time.Time {
	return t.UTC().Truncate(time.Hour)
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

// RunBackfill walks [cursor..UntilUTC] hour by hour (UTC), updating StatePath after each step.
func RunBackfill(ctx context.Context, opts BackfillOptions) error {
	from := truncateHourUTC(opts.FromUTC)
	if from.Before(DefaultBackfillStartUTC) {
		from = DefaultBackfillStartUTC
	}
	until := truncateHourUTC(opts.UntilUTC)
	if until.Before(from) {
		return fmt.Errorf("backfill: until %s before from %s", until.Format(time.RFC3339), from.Format(time.RFC3339))
	}
	if opts.StatePath == "" {
		return fmt.Errorf("backfill: StatePath required")
	}
	maxAttempts := opts.MaxIngestAttempts
	if maxAttempts < 1 {
		maxAttempts = 8
	}

	cursor := from
	if loadedNext, ok, err := loadBackfillState(opts.StatePath); err != nil {
		return err
	} else if ok && loadedNext.After(cursor) {
		cursor = loadedNext
		log.Printf("backfill: resuming from state %s", cursor.Format(time.RFC3339))
	}

	if cursor.After(until) {
		log.Printf("backfill: cursor %s already after until %s, nothing to do", cursor.Format(time.RFC3339), until.Format(time.RFC3339))
		return nil
	}

	log.Printf("backfill: from=%s until=%s (inclusive) state=%s", cursor.Format(time.RFC3339), until.Format(time.RFC3339), opts.StatePath)

	for !cursor.After(until) {
		if err := ctx.Err(); err != nil {
			return err
		}

		url := ghArchiveDataHost + "/" + HourlyArchiveBaseName(cursor)
		exists, err := probeArchiveForBackfill(ctx, url)
		if err != nil {
			return fmt.Errorf("backfill probe %s: %w", url, err)
		}
		if !exists {
			log.Printf("backfill: missing archive (404), skip hour %s", cursor.Format(time.RFC3339))
			next := cursor.Add(time.Hour)
			if err := saveBackfillState(opts.StatePath, next); err != nil {
				return err
			}
			cursor = next
			if err := sleepCtx(ctx, opts.PauseBetween); err != nil {
				return err
			}
			continue
		}

		log.Printf("backfill: ingesting %s", url)
		var ingestErr error
		for attempt := 0; attempt < maxAttempts; attempt++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			ingestErr = IngestDeletingLocal(url)
			if ingestErr == nil {
				break
			}
			log.Printf("backfill: ingest attempt %d/%d failed: %v", attempt+1, maxAttempts, ingestErr)
			if err := sleepCtx(ctx, backoffDuration(attempt)); err != nil {
				return err
			}
		}
		if ingestErr != nil {
			return fmt.Errorf("backfill ingest %s: %w", url, ingestErr)
		}

		next := cursor.Add(time.Hour)
		if err := saveBackfillState(opts.StatePath, next); err != nil {
			return err
		}
		cursor = next
		if err := sleepCtx(ctx, opts.PauseBetween); err != nil {
			return err
		}
	}

	log.Printf("backfill: finished through %s", until.Format(time.RFC3339))
	return nil
}

func backoffDuration(attempt int) time.Duration {
	d := time.Second
	for i := 0; i < attempt && d < 30*time.Second; i++ {
		d *= 2
	}
	return d
}

func sleepCtx(ctx context.Context, d time.Duration) error {
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

// probeArchiveForBackfill returns false,nil for 404; true,nil for 200; retries 429/5xx and network errors.
func probeArchiveForBackfill(ctx context.Context, src string) (bool, error) {
	backoff := time.Second
	for attempt := 0; attempt < 24; attempt++ {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, src, nil)
		if err != nil {
			return false, err
		}
		setArchiveRequestHeaders(req)
		resp, err := hourlyProbeClient.Do(req)
		if err != nil {
			log.Printf("backfill: head %s: %v (retry in %s)", src, err, backoff)
			if err := sleepCtx(ctx, backoff); err != nil {
				return false, err
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		code := resp.StatusCode
		_ = resp.Body.Close()

		switch code {
		case http.StatusOK:
			return true, nil
		case http.StatusNotFound:
			return false, nil
		case http.StatusMethodNotAllowed:
			ok, err := getProbeBackfill(ctx, src)
			if err != nil {
				log.Printf("backfill: get probe %s: %v (retry in %s)", src, err, backoff)
				if err := sleepCtx(ctx, backoff); err != nil {
					return false, err
				}
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			return ok, nil
		case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			log.Printf("backfill: head %s status %d (retry in %s)", src, code, backoff)
			if err := sleepCtx(ctx, backoff); err != nil {
				return false, err
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		default:
			if code >= 500 {
				log.Printf("backfill: head %s status %d (retry in %s)", src, code, backoff)
				if err := sleepCtx(ctx, backoff); err != nil {
					return false, err
				}
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			if code >= 400 {
				return false, fmt.Errorf("head %s: %s", src, resp.Status)
			}
			return false, fmt.Errorf("head %s: unexpected status %d", src, code)
		}
	}
	return false, fmt.Errorf("backfill: probe exhausted retries for %s", src)
}

func getProbeBackfill(ctx context.Context, src string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return false, err
	}
	setArchiveRequestHeaders(req)
	resp, err := hourlyProbeClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		_, _ = io.Copy(io.Discard, resp.Body)
		return true, nil
	case http.StatusNotFound:
		_, _ = io.Copy(io.Discard, resp.Body)
		return false, nil
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		_, _ = io.Copy(io.Discard, resp.Body)
		return false, fmt.Errorf("status %d", resp.StatusCode)
	default:
		if resp.StatusCode >= 500 {
			_, _ = io.Copy(io.Discard, resp.Body)
			return false, fmt.Errorf("status %d", resp.StatusCode)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		if resp.StatusCode >= 400 {
			return false, fmt.Errorf("get %s: %s", src, resp.Status)
		}
		return false, fmt.Errorf("get %s: unexpected %d", src, resp.StatusCode)
	}
}
