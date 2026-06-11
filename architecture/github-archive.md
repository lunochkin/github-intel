# GitHub Archive
From the [GitHub Archive documentation](https://www.gharchive.org/):

GitHub provides 15+ event types, which range from new commits and fork events, to opening new tickets, commenting, and adding members to a project. These events are aggregated into hourly archives, which you can access with any HTTP client:

Query	Command
Activity for 1/1/2015 @ 3PM UTC	wget https://data.gharchive.org/2015-01-01-15.json.gz
Activity for 1/1/2015	wget https://data.gharchive.org/2015-01-01-{0..23}.json.gz
Activity for all of January 2015	wget https://data.gharchive.org/2015-01-{01..31}-{0..23}.json.gz

Each archive contains JSON encoded events as reported by the GitHub API. You can download the raw data and apply own processing to it - e.g. write a custom aggregation script, import it into a database, and so on!

# Comments
Each file is an hourly archive of events.
The `{0..23}` is a bash brace expansion, which expands to `0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23`.

URLs use that **unpadded** hour form (`…-0.json.gz` … `…-9.json.gz`, then `…-10.json.gz`); paths like `…-00.json.gz` are not served (404).

For the manual hourly ingest job (target hour, polling, scope), see [github-hourly-file-subscription.md](github-hourly-file-subscription.md).

For loading history from the first GH Archive hour forward, see [github-archive-backfill.md](github-archive-backfill.md).
