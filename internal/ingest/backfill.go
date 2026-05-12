package ingest

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/errgroup"
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
	// Workers is the number of workers to use for the backfill.
	Workers int
}

func RunBackfill(ctx context.Context, opts BackfillOptions) error {
	tasks := make(chan func() error)

	eg, ctx := errgroup.WithContext(ctx)

	for workerID := range opts.Workers {
		eg.Go(func() error {
			for task := range tasks {
				if err := task(); err != nil {
					return fmt.Errorf("backfill: worker %d: %w", workerID, err)
				}
			}
			return nil
		})
	}

	hours, err := hoursToProcess(ctx, opts)
	if err != nil {
		return err
	}
	maxAttempts := opts.MaxIngestAttempts
	if maxAttempts < 1 {
		maxAttempts = 8
	}

	for cursor := range hours {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case tasks <- func() error {
			if err := addHourUTCInProgress(opts.StatePath, cursor); err != nil {
				return err
			}

			if err := processHour(ctx, cursor, maxAttempts); err != nil {
				return err
			}

			if err := removeHourUTCInProgress(opts.StatePath, cursor); err != nil {
				return err
			}
			if err := sleepOrDone(ctx, opts.PauseBetween); err != nil {
				return err
			}
			return nil
		}:
		}
	}

	close(tasks)
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("backfill: wait: %w", err)
	}

	log.Printf("backfill: finished")
	return nil
}

func hoursToProcess(ctx context.Context, opts BackfillOptions) (chan time.Time, error) {
	if opts.StatePath == "" {
		return nil, fmt.Errorf("backfill: StatePath required")
	}

	from := truncateHourUTC(opts.FromUTC)
	if from.Before(DefaultBackfillStartUTC) {
		from = DefaultBackfillStartUTC
	}

	until := truncateHourUTC(opts.UntilUTC)
	if until.Before(from) {
		return nil, fmt.Errorf("backfill: until %s before from %s", until.Format(time.RFC3339), from.Format(time.RFC3339))
	}

	st, stateOK, err := loadBackfillState(opts.StatePath)
	if err != nil {
		return nil, err
	}
	if !stateOK {
		st = backfillState{}
	}
	ch := make(chan time.Time)

	var nextHour time.Time
	if stateOK && st.NextHourUTC != "" {
		nh, err := parseNextHourUTC(st)
		if err != nil {
			return nil, err
		}
		nextHour = nh
	} else {
		nextHour = from
	}

	cursor := from
	if cursor.Before(nextHour) {
		cursor = nextHour
	}

	if cursor.After(until) {
		log.Printf("backfill: cursor %s already after until %s, nothing to do", cursor.Format(time.RFC3339), until.Format(time.RFC3339))
		close(ch)
		return ch, nil
	}

	go func() {
		defer close(ch)

		for _, hour := range st.HoursUTCInProgress {
			hourTime, err := time.Parse(time.RFC3339, hour)
			if err != nil {
				log.Printf("backfill: invalid hour %s: %v", hour, err)
				continue
			}
			select {
			case <-ctx.Done():
				return
			case ch <- hourTime:
			}
		}

		for !cursor.After(until) {
			select {
			case <-ctx.Done():
				return
			case ch <- cursor:
				cursor = cursor.Add(time.Hour)
			}
		}
	}()

	return ch, nil
}

func processHour(ctx context.Context, cursor time.Time, maxAttempts int) error {
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
		return nil
	}

	log.Printf("backfill: ingesting %s", url)
	var ingestErr error
	for attempt := range maxAttempts {
		if err := ctx.Err(); err != nil {
			return err
		}
		ingestErr = IngestDeletingLocal(url)
		if ingestErr == nil {
			break
		}
		if errors.Is(ingestErr, ErrParse) {
			return fmt.Errorf("backfill ingest %s: %w", url, ingestErr)
		}
		log.Printf("backfill: ingest attempt %d/%d failed: %v", attempt+1, maxAttempts, ingestErr)
		if err := sleepOrDone(ctx, backoffDuration(attempt)); err != nil {
			return err
		}
	}
	if ingestErr != nil {
		return fmt.Errorf("backfill ingest %s: %w", url, ingestErr)
	}
	return nil
}

func backoffDuration(attempt int) time.Duration {
	d := time.Second
	for i := 0; i < attempt && d < 30*time.Second; i++ {
		d *= 2
	}
	return d
}
