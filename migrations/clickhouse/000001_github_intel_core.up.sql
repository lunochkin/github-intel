CREATE DATABASE IF NOT EXISTS github_intel;

CREATE TABLE IF NOT EXISTS github_intel.raw_github_events
(
    `event_id` String,
    `event_type` LowCardinality(String),
    `created_at` DateTime64(3, 'UTC'),
    `actor_id` Int64,
    `actor_login` LowCardinality(String),
    `repo_id` Int64,
    `repo_name` LowCardinality(String),
    `org_id` Nullable(Int64),
    `org_login` Nullable(String),
    `public` UInt8,
    `payload_json` String CODEC(ZSTD(1)),
    `ingested_at` DateTime64(3, 'UTC') DEFAULT now64(3)
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(created_at)
ORDER BY (repo_id, created_at, event_id)
SETTINGS index_granularity = 8192;
