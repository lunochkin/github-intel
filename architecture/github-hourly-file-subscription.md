# Subscribe to GitHub Archive hourly file

Background on archive URLs and format: [github-archive.md](github-archive.md).

There is no specific API to subscribe to GitHub Archive hourly file.

**Subscription here means emulation:** we treat “subscribe” as *wait until the next hourly dump exists, then pull and ingest it*. There is no webhook, feed, or push notification—only HTTP—so the job **polls** on a schedule (coarser during most of the hour, tighter near the hour boundary) until the file is available, which is the practical substitute for being notified when a new archive is published.

We can implement a **manually started** job (no cron or scheduler in scope for now) that downloads **one** hourly archive and ingests it into the database.

## Scope (explicit)
- **Timezone:** All hour arithmetic uses **UTC**, matching GH Archive URLs (`YYYY-MM-DD-H.json.gz`).
- **Target archive:** The **closest future** hourly file relative to job start — i.e. the archive for the **current UTC wall-clock hour** (`truncate(now, hour)`). That file is still being collected for most of the hour and usually **does not exist** on `data.gharchive.org` until shortly **after** that hour ends; the job waits and then ingests it once it appears. It is not the previous hour’s file (`truncate(now, hour) - 1 hour`).
- **No catch-up:** A single run ingests **at most that one** file. Missing runs are not backfilled in this phase.
- **No persisted cursor:** The job does not read a “last ingested file” or other watermark. The target hour is derived **only from `now`**. If you run again before the UTC hour rolls forward, you poll for the same filename again; rely on idempotent ingestion if needed.
- **Trigger:** Start the process manually (CLI, etc.). Overlapping runs are out of scope unless you add a lock later.

## Behaviour
1. Take **current time in UTC** and compute the **target** filename (`…/YYYY-MM-DD-H.json.gz`) for **`truncate(now, hour)`** — the in-progress hour whose dump is the next to be published.
2. If the archive is not yet available (e.g. HTTP 404), **poll** until it is or until a defined give-up rule (implementation waits up to a few hours after that hour’s start; see code).
3. **Polling intervals** while waiting, in terms of the **current UTC clock hour**:
   - **1 minute** from the start of that UTC hour until **2 minutes before** the end of that same UTC hour.
   - **10 seconds** for the **last 2 minutes** of that UTC hour (through the hour boundary).
   - Often the target file appears in the **next** UTC hour; the same interval rule applies hour-by-hour until success or timeout.
4. When the download succeeds, ingest into the database, then **exit** successfully.

*Future work (not required now):* cron-style scheduling, multi-hour catch-up, a stored watermark “after last success,” and overlap locks.
