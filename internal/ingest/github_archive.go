package ingest

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func newGitHubArchiveProbeClient() *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.MaxConnsPerHost = 10
	tr.MaxIdleConnsPerHost = 10
	tr.IdleConnTimeout = 90 * time.Second
	return &http.Client{Timeout: 30 * time.Second, Transport: tr}
}

var gitHubArchiveProbeClient = newGitHubArchiveProbeClient()

const ghArchiveDataHost = "https://data.gharchive.org"

func gitHubArchiveURL(basename string) string {
	return ghArchiveDataHost + "/" + basename
}

// probeGitHubArchive returns false,nil for 404; true,nil for 200; retries 429/5xx and network errors.
func probeGitHubArchive(ctx context.Context, src string) (bool, error) {
	backoff := time.Second
	for range 24 {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, src, nil)
		if err != nil {
			return false, err
		}
		setGitHubArchiveRequestHeaders(req)
		resp, err := gitHubArchiveProbeClient.Do(req)
		if err != nil {
			log.Printf("head %s: %v (retry in %s)", src, err, backoff)
			if err := sleepOrDone(ctx, backoff); err != nil {
				return false, err
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		code := resp.StatusCode
		_ = resp.Body.Close()

		switch code {
		case http.StatusOK:
			return true, nil
		case http.StatusNotFound:
			return false, nil
		case http.StatusMethodNotAllowed:
			ok, err := getProbeBackfill(ctx, src)
			if err != nil {
				log.Printf("backfill: get probe %s: %v (retry in %s)", src, err, backoff)
				if err := sleepOrDone(ctx, backoff); err != nil {
					return false, err
				}
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			return ok, nil
		case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			log.Printf("head %s status %d (retry in %s)", src, code, backoff)
			if err := sleepOrDone(ctx, backoff); err != nil {
				return false, err
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		default:
			if code >= 500 {
				log.Printf("head %s status %d (retry in %s)", src, code, backoff)
				if err := sleepOrDone(ctx, backoff); err != nil {
					return false, err
				}
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			if code >= 400 {
				return false, fmt.Errorf("head %s: %s", src, resp.Status)
			}
			return false, fmt.Errorf("head %s: unexpected status %d", src, code)
		}
	}
	return false, fmt.Errorf("probe exhausted retries for %s", src)
}

func getProbeBackfill(ctx context.Context, src string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return false, err
	}
	setGitHubArchiveRequestHeaders(req)
	resp, err := gitHubArchiveProbeClient.Do(req)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("github archive: close body: %v", err)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		_, _ = io.Copy(io.Discard, resp.Body)
		return true, nil
	case http.StatusNotFound:
		_, _ = io.Copy(io.Discard, resp.Body)
		return false, nil
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		_, _ = io.Copy(io.Discard, resp.Body)
		return false, fmt.Errorf("status %d", resp.StatusCode)
	default:
		if resp.StatusCode >= 500 {
			_, _ = io.Copy(io.Discard, resp.Body)
			return false, fmt.Errorf("status %d", resp.StatusCode)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		if resp.StatusCode >= 400 {
			return false, fmt.Errorf("get %s: %s", src, resp.Status)
		}
		return false, fmt.Errorf("get %s: unexpected %d", src, resp.StatusCode)
	}
}

func downloadGitHubArchiveFile(spec string) error {
	dest, err := localArchivePath(spec)
	if err != nil {
		return fmt.Errorf("local archive path: %w", err)
	}
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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("github archive: close body: %v", err)
		}
	}()

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

func setGitHubArchiveRequestHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "github-intel-ingestor/1.0")
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}
