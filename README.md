# GitHub Engineering Intelligence Backend

## Core idea

Build a backend system that turns public GitHub event data into analytical insights about repositories, organizations, contributors, and engineering activity.

Conceptually:

```text
GitHub public events
        ↓
Ingestion
        ↓
Normalization
        ↓
Aggregation
        ↓
ClickHouse / Postgres
        ↓
Analytical APIs
```

## Main goal

Train production backend/system skills:

* Go backend engineering
* event ingestion
* queues/workers
* ClickHouse analytics
* PostgreSQL metadata storage
* retries/idempotency
* batching/backpressure
* observability
* API design
* performance profiling

## Data sources

Start with:

* GitHub Archive hourly JSON files
* GitHub REST API for enrichment later

Avoid scraping.

## MVP domain

Entities:

```text
repository
organization
user
event
pull_request
issue
release
language
topic
```

Event types:

```text
PushEvent
PullRequestEvent
IssuesEvent
WatchEvent
ForkEvent
CreateEvent
ReleaseEvent
```

## Architecture v1

Start simple:

```text
ingestor-service
        ↓
RabbitMQ / Redpanda
        ↓
worker-service
        ↓
ClickHouse
        ↓
analytics-api
```

Postgres stores metadata:

```text
repositories
organizations
users
import_jobs
api_keys
saved_queries
```

ClickHouse stores analytics data:

```text
raw_github_events
repo_daily_metrics
org_daily_metrics
contributor_daily_metrics
```

## MVP endpoints

```text
GET /repos/:owner/:repo/summary
GET /repos/:owner/:repo/timeseries
GET /repos/:owner/:repo/contributors
GET /repos/:owner/:repo/pr-latency
GET /orgs/:org/velocity
GET /trending/repos?language=go&period=7d
GET /languages/trends?period=30d
```

## Metrics to compute

Repository-level:

```text
stars_per_day
forks_per_day
pushes_per_day
prs_opened
prs_merged
issues_opened
issues_closed
release_count
active_contributors
```

Engineering velocity:

```text
median_pr_review_time
median_pr_merge_time
pr_throughput
issue_resolution_time
release_frequency
contributor_growth
```

Trend metrics:

```text
trending_repositories
fastest_growing_repos
language_growth
org_activity_rankings
```

## Development phases

### Phase 1 — Local ingestion

Goal:

```text
Download one GitHub Archive hourly file
Parse events
Insert raw events into ClickHouse
Expose one summary endpoint
```

Deliverable:

```text
GET /repos/:owner/:repo/summary
```

### Phase 2 — Queue + workers

Goal:

```text
Ingest multiple hourly files
Publish events to queue
Batch insert into ClickHouse
Track import job status in Postgres
```

Practice:

```text
worker pools
context cancellation
batching
retries
idempotency
graceful shutdown
```

### Phase 3 — Analytical queries

Goal:

```text
Build timeseries and ranking endpoints
```

Endpoints:

```text
GET /repos/:owner/:repo/timeseries
GET /trending/repos
GET /languages/trends
```

Practice:

```text
ClickHouse ordering keys
partitions
GROUP BY optimization
materialized views
query profiling
```

### Phase 4 — GitHub API enrichment

Goal:

```text
Enrich repositories with language, topics, description, stars, forks
```

Practice:

```text
rate limits
API clients
retry policies
caching
metadata sync
```

### Phase 5 — Observability

Add:

```text
structured logs
Prometheus metrics
OpenTelemetry traces
pprof
health/readiness endpoints
queue lag metrics
ClickHouse insert latency metrics
```

### Phase 6 — Performance and scaling

Goal:

```text
Process N million events
Measure throughput
Find bottlenecks
Optimize batch size, workers, schema, queries
```

Tools:

```text
pprof
EXPLAIN
ClickHouse system.query_log
load generator
```

## Suggested tech stack

```text
Go
PostgreSQL
ClickHouse
RabbitMQ or Redpanda
Docker Compose
OpenTelemetry
Prometheus
Grafana
GitLab CI or GitHub Actions
```

For AppMagic alignment, RabbitMQ is closer to the vacancy.
For your event-log interests, Redpanda is also fine.

## First concrete task

Build this first:

```text
Docker Compose:
- Go API service
- ClickHouse
- Postgres

Feature:
- download one GH Archive file
- parse PushEvent / PullRequestEvent / WatchEvent
- insert into ClickHouse
- expose GET /repos/:owner/:repo/summary
```

## Anti-overthinking rule

Every week must end with a running increment.

Before each week, write only:

```text
Goal:
What should run by Sunday:
Explicitly not included:
Learning target:
```

No large architecture document before the system works.
