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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lunochkin/github-intel/internal/clickhouse"
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

	ctx := context.Background()
	conn, err := clickhouse.Open(ctx)
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

// IngestDeletingLocal runs Ingest for spec (URL or basename) then deletes the
// locally cached .json.gz if present. Used by backfill to avoid unbounded disk use.
func IngestDeletingLocal(spec string) error {
	if err := Ingest(spec); err != nil {
		return err
	}
	local := localArchivePath(spec)
	if err := os.Remove(local); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove cached archive %s: %w", local, err)
	}
	return nil
}

func loadFile(filename string) (io.ReadCloser, error) {
	local := localArchivePath(filename)
	log.Printf("loading file: %s", local)

	if _, err := os.Stat(local); os.IsNotExist(err) {
		if err := downloadFile(filename); err != nil {
			return nil, fmt.Errorf("download file: %w", err)
		}
	}

	return readFile(local)
}

// localArchivePath returns the filesystem path used for a GH Archive spec.
// HTTP(S) URLs are stored under their path basename in the working directory.
func localArchivePath(spec string) string {
	u, err := url.Parse(spec)
	if err != nil || u.Scheme == "" {
		return spec
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return spec
	}
	base := filepath.Base(u.Path)
	if base == "." || base == "/" {
		return "gharchive.json.gz"
	}
	return base
}

func ghArchiveURL(spec string) string {
	if u, err := url.Parse(spec); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return spec
	}
	return "https://data.gharchive.org/" + filepath.Base(spec)
}

func downloadFile(spec string) error {
	dest := localArchivePath(spec)
	src := ghArchiveURL(spec)
	log.Printf("downloading %s -> %s", src, dest)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "github-intel-ingestor/1.0")
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http get %s: %s", src, resp.Status)
	}

	dir := filepath.Dir(dest)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(dest)+".*.part")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", tmpPath, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		return fmt.Errorf("rename to %s: %w", dest, err)
	}
	committed = true
	return nil
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
			return nil, fmt.Errorf("parse line %d: %w", lineNum, err)
		}
		events = append(events, event)
	}

	return events, nil
}
