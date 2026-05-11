package ingest

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"

	clickhousestd "github.com/ClickHouse/clickhouse-go/v2"
	driverch "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const insertChunkSize = 2000

const insertSQL = `INSERT INTO raw_github_events (
	event_id,
	event_type,
	created_at,
	actor_id,
	actor_login,
	repo_id,
	repo_name,
	org_id,
	org_login,
	public,
	payload_json
)`

func openClickHouse(ctx context.Context) (driverch.Conn, error) {
	host := os.Getenv("CLICKHOUSE_HOST")
	portStr := os.Getenv("CLICKHOUSE_PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return nil, fmt.Errorf("CLICKHOUSE_PORT: invalid %q", portStr)
	}

	opts := &clickhousestd.Options{
		Addr: []string{net.JoinHostPort(host, strconv.Itoa(port))},
		Auth: clickhousestd.Auth{
			Database: os.Getenv("CLICKHOUSE_DATABASE"),
			Username: os.Getenv("CLICKHOUSE_USER"),
			Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		},
	}

	conn, err := clickhousestd.Open(opts)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}
	return conn, nil
}

func nullableOrg(orgID int64, orgLogin string) (id *int64, login *string) {
	if orgID == 0 && orgLogin == "" {
		return nil, nil
	}
	o := orgID
	l := orgLogin
	return &o, &l
}

func insertEventsClickHouse(ctx context.Context, conn driverch.Conn, events []Event) error {
	for start := 0; start < len(events); start += insertChunkSize {
		end := min(start+insertChunkSize, len(events))
		chunk := events[start:end]

		batch, err := conn.PrepareBatch(ctx, insertSQL)
		if err != nil {
			return fmt.Errorf("prepare batch [%d:%d]: %w", start, end, err)
		}
		for _, e := range chunk {
			orgID, orgLogin := nullableOrg(e.OrgID, e.OrgLogin)
			pub := uint8(0)
			if e.Public {
				pub = 1
			}
			if err := batch.Append(
				e.ID,
				e.Type,
				e.CreatedAt.UTC(),
				e.ActorID,
				e.ActorLogin,
				e.RepoID,
				e.RepoName,
				orgID,
				orgLogin,
				pub,
				string(e.Payload),
			); err != nil {
				batch.Abort()
				return fmt.Errorf("append event %q: %w", e.ID, err)
			}
		}
		if err := batch.Send(); err != nil {
			return fmt.Errorf("send batch [%d:%d]: %w", start, end, err)
		}
	}
	return nil
}
