# adapter/snapshot — how this package hangs together

The snapshot runner behind `port/snapshot.Runner`, one per process. A run reads the chosen tables
from one exported database snapshot and hands every row to the processing pipeline as an
`insert` (`prev` nil, timestamp = now, no headers) — the same `database.ProcessMessageHandler` WAL
delivers through, so sinks, operation filters and the operations log see snapshot rows like any
other change.

## The run

1. `Start` validates **every** filter through `database.ValidateWhere` before touching state; one
   bad filter fails the whole request with `ErrInvalidWhere` (422) and nothing starts.
2. Estimate: `GetRowCount` per table; the exact count wins, the rough one otherwise, zero when
   neither arrives. `EstimatedRows` grows to `CompletedRows` if the read overtakes it.
3. `database.StartSnapshot` pauses WAL delivery, locks the tables `ACCESS SHARE` and exports a
   snapshot. Failing here fails every table with the error; there is nothing to finish.
4. Up to `concurrency` (default 10, `WithConcurrency`) tables are read at once via
   `database.ReadSnapshot`. Reads overlap; **`process` calls do not** — `processMu` serializes
   them, because the processing use case's worker pool batch API is not safe for concurrent use.
5. `FinishSnapshot` runs in a defer of the run goroutine, whatever ended the run, which resumes WAL.

Only sinks whose configured operations include `insert` are listed and counted per table.

## Statuses

While processing a table goes `wait → processing → done`. A table that failed mid-run is `done`
with `Error` set (the processing DTO has no error status); `finish` turns `done` without error into
`success`, `done` with error into `error`, and anything unfinished into `error` `"stopped"`.

## Stop, restart, shutdown

- `Stop` turns the state `done` at once and cancels the run; the goroutine releases the snapshot
  afterwards. A `Start` right after waits for that release (`runDone`) before `StartSnapshot` —
  otherwise the idempotent `StartSnapshot` would hand back the dying snapshot's id.
- Every state change from the run goes through `update`, which re-checks the run's context under
  `mu`, so a run that raced `Stop`/`close` never overwrites their state.
- Publishing is throttled to `publishInterval` for per-row progress; status changes always publish.
  Every `/api/live` client is a subscriber too (`stream_updates`), not only an open Snapshot page,
  and each publish clones the state per subscriber under `mu`.
- `New` registers at `StageWorkers`; `close` cancels the run, closes subscriber channels and waits
  for the run goroutine, snapshot release included. A `Start` after that returns `ErrClosed`.

## Metrics

`hump_snapshot_running` (gauge, 0/1) is for external dashboards, not the web UI: it turns 1 in
`Start` and 0 in `finish` (run end, failed `StartSnapshot`, `Stop`) and in `close`, so it matches
the `processing` status, estimation included.
