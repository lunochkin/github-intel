package main

import (
	"context"
	"flag"
	"log"

	"github.com/lunochkin/github-intel/internal/dotenv"
	"github.com/lunochkin/github-intel/internal/ingest"
)

func main() {
	if err := dotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	hourly := flag.Bool("hourly", false, "Poll GH Archive for the current UTC hour's file (next to be published), then ingest once (no positional args)")
	flag.Parse()

	if *hourly {
		if len(flag.Args()) != 0 {
			log.Fatal("usage: ingestor -hourly (no filenames)")
		}
		if err := ingest.RunClosestFutureHourArchive(context.Background()); err != nil {
			log.Fatal(err)
		}
		return
	}

	args := flag.Args()
	log.Printf("args: %v", args)
	if len(args) < 1 {
		log.Fatal("usage: ingestor [-hourly] <filename> [more filenames...]")
	}

	for _, name := range args {
		if err := ingest.Ingest(name); err != nil {
			log.Fatalf("%s: %v", name, err)
		}
	}
}
