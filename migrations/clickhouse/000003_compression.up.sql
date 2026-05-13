ALTER TABLE github_intel.raw_github_events
    MODIFY COLUMN `event_id` String CODEC(ZSTD(3)),
    MODIFY COLUMN `created_at` DateTime64(3, 'UTC') CODEC(DoubleDelta, ZSTD(1)),
    MODIFY COLUMN `actor_id` Int64 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `repo_id` Int64 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `org_id` Nullable(Int64) CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `public` UInt8 CODEC(ZSTD(1)),
    MODIFY COLUMN `payload_json` String CODEC(ZSTD(6)),
    MODIFY COLUMN `ingested_at` DateTime64(3, 'UTC') DEFAULT now64(3) CODEC(DoubleDelta, ZSTD(1));

ALTER TABLE github_intel.repo_daily_metrics
    MODIFY COLUMN `repo_id` Int64 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `day` Date CODEC(DoubleDelta, ZSTD(1)),
    MODIFY COLUMN `stars_delta` Int32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `forks_delta` Int32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `pushes` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `prs_opened` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `prs_merged` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `issues_opened` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `issues_closed` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `releases` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `active_contributors` UInt32 CODEC(T64, ZSTD(1));

ALTER TABLE github_intel.org_daily_metrics
    MODIFY COLUMN `org_id` Int64 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `day` Date CODEC(DoubleDelta, ZSTD(1)),
    MODIFY COLUMN `pushes` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `prs_opened` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `prs_merged` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `issues_opened` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `issues_closed` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `active_repos` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `active_contributors` UInt32 CODEC(T64, ZSTD(1));

ALTER TABLE github_intel.contributor_daily_metrics
    MODIFY COLUMN `repo_id` Int64 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `user_id` Int64 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `day` Date CODEC(DoubleDelta, ZSTD(1)),
    MODIFY COLUMN `commits` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `prs_opened` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `prs_merged` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `issues_opened` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `issues_closed` UInt32 CODEC(T64, ZSTD(1)),
    MODIFY COLUMN `comments` UInt32 CODEC(T64, ZSTD(1));
