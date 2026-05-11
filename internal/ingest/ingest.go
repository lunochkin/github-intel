package ingest

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
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

// String implements fmt.Stringer. Payload is omitted (large; use json.RawMessage parsing when needed).
func (e Event) String() string {
	return fmt.Sprintf(
		"{ID:%v Type:%v CreatedAt:%v ActorID:%v ActorLogin:%v RepoID:%v RepoName:%v OrgID:%v OrgLogin:%v Public:%v, Payload: %s}",
		e.ID, e.Type, e.CreatedAt, e.ActorID, e.ActorLogin, e.RepoID, e.RepoName, e.OrgID, e.OrgLogin, e.Public, string(e.Payload),
	)
}

func Ingest(filename string) error {
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
	if len(events) > 0 {
		log.Printf("events[0]: %v", events[0])
	}

	ctx := context.Background()
	conn, err := openClickHouse(ctx)
	if err != nil {
		return fmt.Errorf("clickhouse: %w", err)
	}
	defer conn.Close()

	if err := insertEventsClickHouse(ctx, conn, events); err != nil {
		return fmt.Errorf("insert clickhouse: %w", err)
	}
	log.Printf("inserted %d rows into raw_github_events", len(events))
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
