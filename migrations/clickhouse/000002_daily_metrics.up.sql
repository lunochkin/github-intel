CREATE TABLE IF NOT EXISTS github_intel.repo_daily_metrics
(
    `repo_id` Int64,
    `repo_name` LowCardinality(String),
    `day` Date,
    `stars_delta` Int32,
    `forks_delta` Int32,
    `pushes` UInt32,
    `prs_opened` UInt32,
    `prs_merged` UInt32,
    `issues_opened` UInt32,
    `issues_closed` UInt32,
    `releases` UInt32,
    `active_contributors` UInt32
)
ENGINE = SummingMergeTree
PARTITION BY toYYYYMM(day)
ORDER BY (repo_id, day);

CREATE TABLE IF NOT EXISTS github_intel.org_daily_metrics
(
    `org_id` Int64,
    `org_login` LowCardinality(String),
    `day` Date,
    `pushes` UInt32,
    `prs_opened` UInt32,
    `prs_merged` UInt32,
    `issues_opened` UInt32,
    `issues_closed` UInt32,
    `active_repos` UInt32,
    `active_contributors` UInt32
)
ENGINE = SummingMergeTree
PARTITION BY toYYYYMM(day)
ORDER BY (org_id, day);

CREATE TABLE IF NOT EXISTS github_intel.contributor_daily_metrics
(
    `repo_id` Int64,
    `user_id` Int64,
    `user_login` LowCardinality(String),
    `day` Date,
    `commits` UInt32,
    `prs_opened` UInt32,
    `prs_merged` UInt32,
    `issues_opened` UInt32,
    `issues_closed` UInt32,
    `comments` UInt32
)
ENGINE = SummingMergeTree
PARTITION BY toYYYYMM(day)
ORDER BY (repo_id, user_id, day);
