package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/lunochkin/github-intel/internal/dotenv"
	"github.com/lunochkin/github-intel/internal/ingest"
)

func main() {
	if err := dotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	hourly := flag.Bool("hourly", false, "Poll GH Archive for the current UTC hour's file (next to be published), then ingest once (no positional args)")
	backfill := flag.Bool("backfill", false, "Sequential historical ingest from -backfill-from through -backfill-until (UTC); uses -backfill-state for resume")
	bfFrom := flag.String("backfill-from", "2011-02-12", "UTC start (YYYY-MM-DD or RFC3339); not before GH Archive start")
	bfUntil := flag.String("backfill-until", "", "UTC last hour to include (YYYY-MM-DD or RFC3339); default last completed hour")
	bfState := flag.String("backfill-state", "data/backfill-state.json", "JSON cursor file for resume")
	bfPause := flag.Duration("backfill-pause", 500*time.Millisecond, "sleep after each ingested or skipped hour")
	bfMaxAttempts := flag.Int("backfill-ingest-retries", 8, "max ingest attempts per hour before failing")
	bfWorkers := flag.Int("backfill-workers", 10, "number of workers to use for the backfill")
	flag.Parse()

	if *hourly && *backfill {
		log.Fatal("use only one of -hourly or -backfill")
	}

	if *hourly {
		if len(flag.Args()) != 0 {
			log.Fatal("usage: ingestor -hourly (no filenames)")
		}
		if err := ingest.RunClosestFutureHourArchive(context.Background()); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *backfill {
		if len(flag.Args()) != 0 {
			log.Fatal("usage: ingestor -backfill (no filenames)")
		}
		from, err := ingest.ParseBackfillInstant(*bfFrom)
		if err != nil {
			log.Fatalf("-backfill-from: %v", err)
		}
		var until time.Time
		if *bfUntil == "" {
			until = time.Now().UTC().Truncate(time.Hour).Add(-time.Hour)
		} else {
			until, err = ingest.ParseBackfillInstant(*bfUntil)
			if err != nil {
				log.Fatalf("-backfill-until: %v", err)
			}
		}
		opts := ingest.BackfillOptions{
			FromUTC:           from,
			UntilUTC:          until,
			StatePath:         *bfState,
			PauseBetween:      *bfPause,
			MaxIngestAttempts: *bfMaxAttempts,
			Workers:           *bfWorkers,
		}
		if err := ingest.RunBackfill(context.Background(), opts); err != nil {
			log.Fatal(err)
		}
		return
	}

	args := flag.Args()
	log.Printf("args: %v", args)
	if len(args) < 1 {
		log.Fatal("usage: ingestor [-hourly | -backfill] <filename> [more filenames...]")
	}

	for _, name := range args {
		if err := ingest.Ingest(name); err != nil {
			log.Fatalf("%s: %v", name, err)
		}
	}
}
