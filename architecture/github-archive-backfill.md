# GitHub Archive backfill

Background: [github-archive.md](github-archive.md). Live “next file” polling: [github-hourly-file-subscription.md](github-hourly-file-subscription.md).

## Purpose

**Backfill** loads historical GH Archive hourly files **from the beginning of the dataset** forward in time, ingesting them into the same store as the subscription-style hourly job. There is still no push API: backfill is **sequential HTTP download + ingest**, controlled by our process.

## Dataset range (UTC)

Per [GH Archive](https://www.gharchive.org/):

- **First archives:** from **2011-02-12** (inclusive), in hourly files named `YYYY-MM-DD-H.json.gz`.
- **2011-02-12 through 2014-12-31:** recorded from the legacy Timeline API.
- **From 2015-01-01:** recorded from the Events API (same URL pattern; event shapes evolved over the years).

The backfill **cursor** should advance in **UTC hour steps** from an explicit **start** (default: `2011-02-12 00:00:00 UTC`) through an explicit **end** (see below).

## Ordering and “gradual” loading

1. **Oldest first:** Walk hours in strict chronological order:  
   `H0 = start truncated to hour`, then `H0 + 1h`, `H0 + 2h`, … until **end**.
2. **One archive per step (or small batches):** Process a single hour file per iteration by default so progress is easy to checkpoint and failures are bounded. Optionally batch *downloads* ahead of time; **commit the cursor only after a successful ingest** for that hour.
3. **Politeness:** Insert a configurable **sleep** between successful ingests (e.g. hundreds of ms to a few seconds) and use **backoff** on HTTP 429 / 5xx so we do not hammer `data.gharchive.org`.
4. **End boundary:** Define `end` as the **last hour to include** in backfill, typically **just before** the range owned by the live hourly job—for example the **last fully completed UTC hour at backfill start**, or `now.Truncate(hour) - 1h` when handing off to “subscription” ingestion. That avoids duplicate work and races on the file that is still “upcoming” for the poller.

## URL and availability

- Canonical file: `https://data.gharchive.org/YYYY-MM-DD-H.json.gz` with `H` in `0..23` (UTC).
- **404:** Treat as “missing for this hour” (gaps can exist in very old data). Log, **skip or quarantine** per policy, and **advance the cursor** so the run does not stall forever. Optionally retry with limit before skipping.
- **Transient errors:** Retry with exponential backoff; do not advance cursor until ingest succeeds (or until skip policy triggers).

## Progress and restart

- **Persist a watermark** (e.g. last successfully ingested hour start as UTC `time.Time`, or `YYYY-MM-DD-H` string). On startup, **resume from the next hour** after the watermark.
- **Idempotency:** Re-ingesting the same hour should be safe (same `event_id`s and `ReplacingMergeTree` / dedup strategy as current ingest).
- **Atomicity:** Update the stored watermark **only after** download + parse + insert succeed for that hour (or after a deliberate skip with audit).

## Resources

- **Disk:** Each hour is a large gzip; download to temp and **delete** after successful ingest.
- **Database:** Large historical runs stress insert rate and merges; tune batch size separately from the live hourly path if needed.
- **Observability:** Log hour, file URL, row count, duration, and errors; optional metrics for lag behind target end.

## Relation to the hourly subscription job

| Concern | Backfill | Hourly subscription (`ingestor -hourly`) |
|--------|----------|---------------------------------------------|
| Direction | Oldest → newer, bounded window | Waits for **current** UTC hour file (next published) |
| Cursor | **Required** (persisted) | Not required (derived from `now`) |
| Polling cadence | Mostly file exists immediately; still use retries/backoff | 1 min / 10 s pattern near hour boundary |
| Overlap | Stop backfill **before** files the poller owns | Start after backfill watermark catches up |

## CLI (`cmd/ingestor`)

Flags (UTC):

- `-backfill` — run sequential backfill (no positional filenames).
- `-backfill-from` — start date (`YYYY-MM-DD` or RFC3339). Default `2011-02-12` (GH Archive start; see above).
- `-backfill-until` — inclusive last hour; default = last completed UTC hour (`now.Truncate(hour) - 1h`).
- `-backfill-state` — cursor JSON path (default `backfill-state.json`). Next hour to process after last success/skip.
- `-backfill-pause` — sleep after each hour (default `500ms`).
- `-backfill-ingest-retries` — per-hour ingest retries (default `8`).

Example:

```bash
go run ./cmd/ingestor -backfill \
  -backfill-from=2011-02-12 \
  -backfill-until=2015-01-01 \
  -backfill-state=./var/backfill-state.json \
  -backfill-pause=1s
```

Or: `make run-ingestor-backfill BACKFILL_ARGS='-backfill-until=2015-01-01'`

## Future work

- Parallel workers over disjoint ranges (with proven idempotency).
- Optional per-hour validation (row counts / checksums).
