package main

import (
	"flag"
	"log"

	"github.com/lunochkin/github-intel/internal/dotenv"
	"github.com/lunochkin/github-intel/internal/ingest"
)

func main() {
	if err := dotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	flag.Parse()
	args := flag.Args()
	log.Printf("args: %v", args)
	if len(args) < 1 {
		log.Fatal("usage: ingestor <filename> [more filenames...]")
	}

	for _, name := range args {
		if err := ingest.Ingest(name); err != nil {
			log.Fatalf("%s: %v", name, err)
		}
	}
}
