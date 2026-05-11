package ingest

import (
	"context"
	"fmt"
	"log"
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

		url := gitHubArchiveURL(HourlyArchiveBaseName(cursor))
		exists, err := probeGitHubArchive(ctx, url)
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
			if err := sleepOrDone(ctx, opts.PauseBetween); err != nil {
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
			if err := sleepOrDone(ctx, backoffDuration(attempt)); err != nil {
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
		if err := sleepOrDone(ctx, opts.PauseBetween); err != nil {
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
