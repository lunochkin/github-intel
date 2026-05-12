package ingest

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/lunochkin/github-intel/internal/clickhouse"
)

// ErrParse marks unrecoverable archive parse failures (malformed JSON,
// unsupported timestamp format). Backfill must not retry these.
var ErrParse = errors.New("parse")

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
	r, err := loadFile(filename)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("load file: close: %v", err)
		}
	}()

	events, err := parse(r)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	log.Printf("parsed %d events", len(events))

	ctx := context.Background()
	conn, err := clickhouse.Open(ctx)
	if err != nil {
		return fmt.Errorf("clickhouse: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("clickhouse: close: %v", err)
		}
	}()

	if err := insertEventsClickHouse(ctx, conn, events); err != nil {
		return fmt.Errorf("insert clickhouse: %w", err)
	}
	log.Printf("inserted %d rows into raw_github_events", len(events))
	return nil
}

// IngestDeletingLocal runs Ingest for spec (URL or basename) then deletes the
// locally cached .json.gz if present. Used by backfill to avoid unbounded disk use.
func IngestDeletingLocal(spec string) error {
	if err := Ingest(spec); err != nil {
		return err
	}
	local, err := localArchivePath(spec)
	if err != nil {
		return fmt.Errorf("local archive path: %w", err)
	}
	if err := os.Remove(local); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove cached archive %s: %w", local, err)
	}
	return nil
}

func loadFile(filename string) (io.ReadCloser, error) {
	local, err := localArchivePath(filename)
	if err != nil {
		return nil, fmt.Errorf("local archive path: %w", err)
	}
	log.Printf("loading file: %s", local)

	if _, err := os.Stat(local); os.IsNotExist(err) {
		if err := downloadGitHubArchiveFile(filename); err != nil {
			return nil, fmt.Errorf("download file: %w", err)
		}
	}

	return readFile(local)
}

// localArchivePath returns the filesystem path used for a GH Archive spec.
// HTTP(S) URLs are stored under data/ using their URL path basename.
func localArchivePath(spec string) (string, error) {
	u, err := url.Parse(spec)
	if err != nil || u.Scheme == "" {
		return spec, nil
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return spec, nil
	}
	base := filepath.Base(u.Path)
	if base == "." || base == "/" {
		return "", fmt.Errorf("invalid spec: %s", spec)
	}
	return filepath.Join("data", base), nil
}

func ghArchiveURL(spec string) string {
	if u, err := url.Parse(spec); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return spec
	}
	return "https://data.gharchive.org/" + filepath.Base(spec)
}

func readFile(filename string) (io.ReadCloser, error) {
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
			return nil, fmt.Errorf("%w line %d: %w", ErrParse, lineNum, err)
		}
		events = append(events, event)
	}

	return events, nil
}
