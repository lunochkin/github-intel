# GitHub Intel

A backend that turns public GitHub event data into analytical insights about
repositories, organizations, and contributors.

> A practice build in public: production-grade backend patterns — ingestion,
> analytics storage, minimal API — exercised on real GitHub event data. Early
> stage; the [Status](#status) section is the source of truth for what works.

Events are pulled from [GH Archive](https://www.gharchive.org/) hourly JSON
files, parsed, and stored in ClickHouse for analytical queries. A small HTTP API
exposes aggregates, with a React dashboard on top.

```text
GH Archive hourly files
        │
        ▼
    ingestor  ──►  ClickHouse  ──►  api  ──►  web dashboard
```

## Status

Early stage. What works today:

- GH Archive download with atomic temp-file-then-rename, retries, and GOAWAY
  handling.
- Event parsing with legacy-format tolerance (old slash-format `created_at`,
  numeric `public`).
- Backfill: sequential historical ingest with a resumable cursor; hourly mode
  polls for the latest published file.
- Raw events inserted into ClickHouse.
- `GET /summary` — global aggregate over all ingested events (row count, unique
  events/repos/actors, time range).
- Minimal React/Vite dashboard.

Not yet built: typed per-event-type parsing, scoped per-repo/org endpoints,
queue + workers, Postgres metadata, observability, GitHub API enrichment. See
[Roadmap](#roadmap).

## Tech stack

- **Go** — ingestor and HTTP API
- **ClickHouse** — analytics storage (migrations via golang-migrate)
- **React + TypeScript + Vite** — dashboard (`web/`)

## Getting started

### Prerequisites

- Go (see `go.mod` for the version)
- A running ClickHouse instance
- Node.js (for the web UI)

### Configure

Copy the example env and adjust:

```sh
cp .env.example .env
```

| Variable                         | Default          | Purpose                                  |
| -------------------------------- | ---------------- | ---------------------------------------- |
| `LISTEN_ADDR`                    | `:8800`          | API listen address                       |
| `CLICKHOUSE_HOST`                | `localhost`      | ClickHouse host                          |
| `CLICKHOUSE_PORT`                | `9000`           | Native protocol port                     |
| `CLICKHOUSE_USER`                | `default`        | ClickHouse user                          |
| `CLICKHOUSE_PASSWORD`            | _(empty)_        | ClickHouse password                      |
| `CLICKHOUSE_DATABASE`            | `github_intel`   | Database for reads/writes                |
| `CLICKHOUSE_MIGRATIONS_DATABASE` | `default`        | Where `schema_migrations` is stored      |

### Run migrations

```sh
make run-migrate-clickhouse           # apply all (up)
make run-migrate-clickhouse MIGRATE_ARGS=down
```

### Ingest data

```sh
# Ingest a specific local file
make run-ingestor INGESTOR_ARGS=data/2015-01-01-0.json.gz

# Poll and ingest the latest published hourly file
make run-ingestor-hourly

# Sequential historical backfill (resumable)
make run-ingestor-backfill BACKFILL_ARGS='-backfill-from=2015-01-01 -backfill-until=2015-01-02'
```

### Run the API

```sh
make run-api        # http://localhost:8800
make dev-api        # with file-watch reload
```

Endpoints:

- `GET /summary` — aggregate statistics
- `GET /healthz` — health check

### Run the dashboard

```sh
make web-install
make web-dev        # Vite dev server, proxies /summary and /healthz to :8800
make web-build      # production build into web/dist
```

## Development

```sh
make build          # compile binaries into bin/
make test           # go test -short ./...
make test-race      # go test -race ./...
make vet
make lint           # requires golangci-lint
make fmt
make tidy
```

Design notes live in [`architecture/`](architecture/).

## Roadmap

- Typed parsing for `PushEvent` / `PullRequestEvent` / `IssuesEvent` / etc.
- Scoped endpoints: `/repos/:owner/:repo/summary`, timeseries, contributors,
  PR latency; `/orgs/:org/velocity`; `/trending/repos`; `/languages/trends`.
- Queue + worker pools, batched inserts, import-job tracking in Postgres,
  idempotency, graceful shutdown.
- Observability: structured logs, Prometheus metrics, OpenTelemetry traces,
  pprof, health/readiness.
- GitHub REST API enrichment (languages, topics, stars) with rate limiting.
- Performance work: profiling, schema/query tuning, throughput measurement.

## Data source

Built on [GH Archive](https://www.gharchive.org/), which publishes the public
GitHub event stream as hourly JSON files. No scraping.

## License

[MIT](LICENSE)
</content>
</invoke>
