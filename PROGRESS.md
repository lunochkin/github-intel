# Project Progress

## Initial project scope

A 6-phase backend project in Go to turn public GitHub event data into analytical insights.

**Phase 1** — Local ingestion: download one GH Archive file, parse events, insert into ClickHouse, expose one summary endpoint.

**Phase 2** — Queue + workers: multiple hourly files, RabbitMQ/Redpanda, batch inserts, import job tracking in Postgres, retries, idempotency, graceful shutdown.

**Phase 3** — Analytical queries: timeseries endpoint, trending repos, language trends, ClickHouse query optimization.

**Phase 4** — GitHub API enrichment: rate limits, retry policies, metadata sync.

**Phase 5** — Observability: structured logs, Prometheus, OpenTelemetry, pprof, health endpoints.

**Phase 6** — Performance and scaling: profiling, bottleneck analysis, throughput measurement.

---

## What's actually done

Phase 1 is partially complete. Specifically:

**Done:**
- Project scaffolding, Docker Compose, `.env` loading
- ClickHouse migrations: `raw_github_events`, `repo_daily_metrics`, `org_daily_metrics`, `contributor_daily_metrics` (3 migration files including compression codecs)
- GitHub Archive download (with temp-file-then-rename atomicity, retry logic, GOAWAY handling)
- Event parsing with legacy format tolerance (`created_at` as old GH date string, `public` as 0/1)
- Parallel processing during ingestion
- Backfill: subscribing to hourly files, tracking backfill state, deleting local files after insert
- ClickHouse insert of raw events
- A global `GET /summary` endpoint — returns aggregate counts across all data (total events, unique repos, unique actors, time range)
- A minimal web UI (Vite/React scaffolded, has a simple dashboard)

**Not done yet (from Phase 1's own checklist):**
- Event-type filtering: `PushEvent`, `PullRequestEvent`, `WatchEvent` are not parsed into typed structs — everything goes in as raw JSON payload
- `GET /repos/:owner/:repo/summary` — the scoped per-repo endpoint does not exist; the current `/summary` is a global aggregate

**Phase 2 and beyond:** not started. No queue, no Postgres, no job tracking, no workers, no observability, no enrichment.

---

In short: the ingestion pipeline runs end-to-end and data lands in ClickHouse, but the API layer is a stub (one global count endpoint), and the project is still within the first phase.
