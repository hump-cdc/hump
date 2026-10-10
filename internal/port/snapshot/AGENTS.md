# port/snapshot — how this package hangs together

The snapshot run as the web UI sees it: `Request` going in, `State` streaming out, and the
`Runner` that owns one run at a time. One file, `snapshot.go`, importing `domain` and the
standard library. `adapter/snapshot` is the only implementation; how a run actually proceeds is
in `adapter/snapshot/AGENTS.md`.

Not to be confused with `metricscollector.Snapshot` (one metrics dump) or the database snapshot
of `port/database` (`StartSnapshot` / `ReadSnapshot`), which the adapter drives.

## Who calls what

`start_snapshot` → `Start`, `stop_snapshot` → `Stop`, `stream_snapshot` → `Subscribe` /
`Unsubscribe`. **Nil is a valid `Runner`**: it is wired only when the web UI is enabled and the
process is not in cleanup mode, and all three use cases nil-check it.

## `State` is sent whole, and lossily

Every change publishes the complete `State`, never a diff, and a subscriber that falls behind
gets the latest one rather than every one. A consumer must therefore render from the state it
holds and never accumulate across messages. `Subscribe` returns the current state together with
the channel, so there is no gap between reading and subscribing; a closed runner answers a
closed channel.

## The status literals are wire formats

`stream_snapshot/dto.go` repeats them as `enum` tags, which nothing derives from the constants:
`idle`, `processing`, `done` for `Status` (one response type each), and two disjoint sets for
`TableStatus` — `wait,processing,done` while the run is processing, `success,error` once it is
done. Adding or renaming a constant means editing those tags.

That split is the thing to keep straight: `TableDone` during a run means "read to the end or
failed" — there is no error status while processing, so `Table.Error` is what tells the two
apart. Once the run is done, `done` without an error becomes `success`, `done` with one becomes
`error`, and a table the run never finished ends as `error` too.

## `Stop` does two things

It ends a run in progress, **or** clears a finished one back to `idle` — the UI's "dismiss".
Only an already idle runner answers `ErrNotRunning`.

## Errors

| Sentinel | From | HTTP |
|---|---|---|
| `ErrAlreadyRunning` | `Start` | 409 in `start_snapshot` |
| `ErrInvalidWhere` | `Start` | 422 in `start_snapshot` |
| `ErrNotRunning` | `Stop` | 409 in `stop_snapshot` |
| `ErrClosed` | `Start` after shutdown | not mapped — falls through as a plain error |

`ErrInvalidWhere` here is a different value from `port/database.ErrInvalidWhere` with the same
message. The adapter joins the database's error into this one, so a controller matches this
package's sentinel and still shows the parser's text. `Start` validates every filter before
anything starts: one bad filter fails the whole request.

## `Sink` here is a progress row

`snapshot.Sink` is `(Type, Host, Target, Count)` for display, with plain strings — unrelated to
the `port/sink.Sink` interface. Only sinks whose operations include `insert` appear, since a
snapshot row is delivered as an insert.
