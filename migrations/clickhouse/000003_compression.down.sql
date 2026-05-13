ALTER TABLE github_intel.raw_github_events
    MODIFY COLUMN `event_id` String,
    MODIFY COLUMN `created_at` DateTime64(3, 'UTC'),
    MODIFY COLUMN `actor_id` Int64,
    MODIFY COLUMN `repo_id` Int64,
    MODIFY COLUMN `org_id` Nullable(Int64),
    MODIFY COLUMN `public` UInt8,
    MODIFY COLUMN `payload_json` String CODEC(ZSTD(1)),
    MODIFY COLUMN `ingested_at` DateTime64(3, 'UTC') DEFAULT now64(3);

ALTER TABLE github_intel.repo_daily_metrics
    MODIFY COLUMN `repo_id` Int64,
    MODIFY COLUMN `day` Date,
    MODIFY COLUMN `stars_delta` Int32,
    MODIFY COLUMN `forks_delta` Int32,
    MODIFY COLUMN `pushes` UInt32,
    MODIFY COLUMN `prs_opened` UInt32,
    MODIFY COLUMN `prs_merged` UInt32,
    MODIFY COLUMN `issues_opened` UInt32,
    MODIFY COLUMN `issues_closed` UInt32,
    MODIFY COLUMN `releases` UInt32,
    MODIFY COLUMN `active_contributors` UInt32;

ALTER TABLE github_intel.org_daily_metrics
    MODIFY COLUMN `org_id` Int64,
    MODIFY COLUMN `day` Date,
    MODIFY COLUMN `pushes` UInt32,
    MODIFY COLUMN `prs_opened` UInt32,
    MODIFY COLUMN `prs_merged` UInt32,
    MODIFY COLUMN `issues_opened` UInt32,
    MODIFY COLUMN `issues_closed` UInt32,
    MODIFY COLUMN `active_repos` UInt32,
    MODIFY COLUMN `active_contributors` UInt32;

ALTER TABLE github_intel.contributor_daily_metrics
    MODIFY COLUMN `repo_id` Int64,
    MODIFY COLUMN `user_id` Int64,
    MODIFY COLUMN `day` Date,
    MODIFY COLUMN `commits` UInt32,
    MODIFY COLUMN `prs_opened` UInt32,
    MODIFY COLUMN `prs_merged` UInt32,
    MODIFY COLUMN `issues_opened` UInt32,
    MODIFY COLUMN `issues_closed` UInt32,
    MODIFY COLUMN `comments` UInt32;
