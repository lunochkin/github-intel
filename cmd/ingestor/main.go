package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/lunochkin/github-intel/internal/dotenv"
)

type Event struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	CreatedAt  time.Time       `json:"created_at"`
	ActorID    int64           `json:"actor_id"`
	ActorLogin string          `json:"actor_login"`
	RepoID     int64           `json:"repo_id"`
	RepoName   string          `json:"repo_name"`
	OrgID      int64           `json:"org_id"`
	OrgLogin   string          `json:"org_login"`
	Public     bool            `json:"public"`
	Payload    json.RawMessage `json:"payload"`
}

func main() {
	if err := dotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	flag.Parse()
	args := flag.Args()
	log.Printf("args: %v", args)
	if len(args) < 1 {
		log.Fatal("usage: ingestor <filename>")
	}
	name := args[0]

	if err := run(name); err != nil {
		log.Fatal(err)
	}
}

func run(filename string) error {
	// TODO: Implement download

	r, err := loadFile(filename)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	defer r.Close()

	events, err := parse(r)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	log.Printf("parsed %d events", len(events))
	log.Printf("events[0]: %+v", events[0])

	// TODO: Implement load into ClickHouse
	return nil
}

func loadFile(filename string) (io.ReadCloser, error) {
	bodyCompressed, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(bodyCompressed))
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	return gzipReader, nil
}

func parse(r io.Reader) ([]Event, error) {
	events := make([]Event, 0)

	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("parse line %d: %w", lineNum, err)
		}
		events = append(events, event)
	}

	return events, nil
}
