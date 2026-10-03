# adapter/snapshot_mock — how this package hangs together

A **stand-in** for the real snapshot runner behind `port/snapshot.Runner`. It reads no database and
delivers to no sink: `Start` invents a row count and sinks per table (derived from a hash of
`schema.name`, so a table looks the same on every run), ignores the filters, and a goroutine moves
the run forward every tick. It exists so the web UI's snapshot screen has a backend to drive. When
the real runner lands it implements the same port, `cmd/hump/main.go` hands it in instead, and this
package goes.

## The run

- One run at a time. `Start` while processing is `ErrAlreadyRunning`; from idle or done it starts over.
- Each tick at most `concurrentTables` tables are read, `rowsPerTick` rows of each per tick.
  Every tick publishes. When every table is done the run turns `done` and its tables `success`.
- `Stop` turns the run `done` at once: finished tables `success`, the rest `error` with `"stopped"`.
  `Stop` without a run in progress is `ErrNotRunning`.
- `EstimatedRows` is cleared once done; the done state does not carry it.

## Publishing

Every state is complete, so a subscriber only ever needs the latest one: channels hold one slot,
and `publish` drains a state the subscriber has not taken before sending the new one. That
drain-and-send is race-free only because it runs under `mu`, which every sender holds.

`Stop` and `close` cancel the run under `mu`, and the run goroutine re-checks its context under
`mu` before advancing, so a tick that raced them never overwrites their state.

## Shutdown

`New` registers at `StageWorkers`, like `live_streamer`, so subscriber channels close — and SSE
handlers return through their `!ok` branch — before `StageServers` shuts the web server down.
`close` is idempotent and waits for the run goroutine.
