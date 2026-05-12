package ingest

import (
	"context"
	"fmt"

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
				if err := batch.Abort(); err != nil {
					return fmt.Errorf("batch abort: %w", err)
				}
				return fmt.Errorf("append event %q: %w", e.ID, err)
			}
		}
		if err := batch.Send(); err != nil {
			return fmt.Errorf("send batch [%d:%d]: %w", start, end, err)
		}
	}
	return nil
}
