package summary

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	driverch "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const querySQL = `
SELECT
	count() AS event_rows,
	uniqExact(event_id) AS unique_events,
	uniqExact(repo_id) AS unique_repos,
	uniqExact(actor_id) AS unique_actors,
	min(created_at) AS first_event_at,
	max(created_at) AS last_event_at
FROM raw_github_events
`

type response struct {
	EventRows    uint64     `json:"event_rows"`
	UniqueEvents uint64     `json:"unique_events"`
	UniqueRepos  uint64     `json:"unique_repos"`
	UniqueActors uint64     `json:"unique_actors"`
	FirstEventAt *time.Time `json:"first_event_at,omitempty"`
	LastEventAt  *time.Time `json:"last_event_at,omitempty"`
}

type jsonErr struct {
	Error string `json:"error"`
}

// Handler serves GET summary statistics from raw_github_events.
func Handler(conn driverch.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		row := conn.QueryRow(ctx, querySQL)
		var (
			eventRows, uniqueEvents, uniqueRepos, uniqueActors uint64
			first, last                                        time.Time
		)
		if err := row.Scan(&eventRows, &uniqueEvents, &uniqueRepos, &uniqueActors, &first, &last); err != nil {
			log.Printf("summary: scan: %v", err)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(jsonErr{Error: "failed to load summary"})
			return
		}

		resp := response{
			EventRows:    eventRows,
			UniqueEvents: uniqueEvents,
			UniqueRepos:  uniqueRepos,
			UniqueActors: uniqueActors,
		}
		if eventRows > 0 {
			f := first.UTC()
			l := last.UTC()
			resp.FirstEventAt = &f
			resp.LastEventAt = &l
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(resp); err != nil {
			log.Printf("summary: encode: %v", err)
		}
	}
}
