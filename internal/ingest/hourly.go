package ingest

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const ghArchiveDataHost = "https://data.gharchive.org"

var hourlyProbeClient = &http.Client{Timeout: 60 * time.Second}

// giveUpAfterTargetStart is how long after the target archive hour begins we stop
// polling (archive for hour H is usually published during H+1; H+3 is a safe cap).
const giveUpAfterTargetStart = 3 * time.Hour

// ClosestFutureArchiveHourUTC returns the UTC start of the current wall-clock hour
// (truncate(now, hour)). That hour’s `.json.gz` is the closest archive file not yet
// published while the hour is in progress; it typically becomes available shortly
// after this instant + 1h.
func ClosestFutureArchiveHourUTC(now time.Time) time.Time {
	return now.UTC().Truncate(time.Hour)
}

// HourlyArchiveBaseName returns the GH Archive filename for the hour starting at t (UTC).
// The hour is decimal without a leading zero for 0–9 (e.g. ...-0.json.gz), matching data.gharchive.org;
// padded forms such as ...-00.json.gz return 404.
func HourlyArchiveBaseName(hourStartUTC time.Time) string {
	t := hourStartUTC.UTC()
	return fmt.Sprintf("%s-%d.json.gz", t.Format("2006-01-02"), t.Hour())
}

func hourlyArchiveURL(basename string) string {
	return ghArchiveDataHost + "/" + basename
}

// pollInterval returns the sleep duration before the next poll attempt, using the
// current UTC wall clock (see architecure/github-hourly-file-subscription.md).
func pollInterval(now time.Time) time.Duration {
	now = now.UTC()
	hourStart := now.Truncate(time.Hour)
	endOfHour := hourStart.Add(time.Hour)
	twoMinBeforeEnd := endOfHour.Add(-2 * time.Minute)
	if now.Before(twoMinBeforeEnd) {
		return time.Minute
	}
	return 10 * time.Second
}

// RunClosestFutureHourArchive polls until the archive for ClosestFutureArchiveHourUTC(now)
// is available, then ingests it once (same semantics as Ingest with an HTTPS URL).
func RunClosestFutureHourArchive(parent context.Context) error {
	targetHour := ClosestFutureArchiveHourUTC(time.Now())
	basename := HourlyArchiveBaseName(targetHour)
	src := hourlyArchiveURL(basename)
	log.Printf("hourly job: target UTC hour start %s, file %s", targetHour.Format(time.RFC3339), basename)

	deadline := targetHour.Add(giveUpAfterTargetStart)
	ctx, cancel := context.WithDeadline(parent, deadline)
	defer cancel()

	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("gh archive hourly: %w (target %s)", err, basename)
		}

		ok, err := archiveAvailable(ctx, src)
		if err != nil {
			return err
		}
		if ok {
			log.Printf("hourly job: archive available, ingesting %s", src)
			return Ingest(src)
		}

		wait := pollInterval(time.Now().UTC())
		log.Printf("hourly job: %s not ready, retry in %s", basename, wait)
		if err := sleepOrDone(ctx, wait); err != nil {
			return fmt.Errorf("gh archive hourly: %w (target %s)", err, basename)
		}
	}
}

func sleepOrDone(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func archiveAvailable(ctx context.Context, src string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, src, nil)
	if err != nil {
		return false, fmt.Errorf("build head: %w", err)
	}
	setArchiveRequestHeaders(req)

	resp, err := hourlyProbeClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("head %s: %w", src, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusMethodNotAllowed:
		return archiveAvailableGET(ctx, src)
	default:
		if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			log.Printf("hourly job: head %s: %s (treating as not ready, will retry)", src, resp.Status)
			return false, nil
		}
		return false, fmt.Errorf("head %s: %s", src, resp.Status)
	}
}

func archiveAvailableGET(ctx context.Context, src string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return false, fmt.Errorf("build get: %w", err)
	}
	setArchiveRequestHeaders(req)

	resp, err := hourlyProbeClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("get %s: %w", src, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		_, _ = io.Copy(io.Discard, resp.Body)
		return true, nil
	case http.StatusNotFound:
		_, _ = io.Copy(io.Discard, resp.Body)
		return false, nil
	default:
		_, _ = io.Copy(io.Discard, resp.Body)
		if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			log.Printf("hourly job: get %s: %s (treating as not ready, will retry)", src, resp.Status)
			return false, nil
		}
		return false, fmt.Errorf("get %s: %s", src, resp.Status)
	}
}

func setArchiveRequestHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "github-intel-ingestor/1.0")
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}
