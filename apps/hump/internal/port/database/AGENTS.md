# port/database — how this package hangs together

The CDC source as the rest of the process sees it: one `Database` that streams row changes,
describes and counts tables, and reads them through an exported snapshot. `adapter/postgres` is
the only implementation; its mechanics are in `adapter/postgres/AGENTS.md`.

## One wide interface, on purpose

`Database` is not split per consumer. `cmd/hump/main.go` builds one value and hands the whole
interface to every consumer; each uses the few methods it needs. Nothing
asserts `var _ Database = (*T)(nil)` — the typed `var db databasePort.Database = …` in `main.go`
is the only compile-time proof. The test fakes (`fakeDatabase` in `adapter/snapshot`,
`processing/cleanup_database` and the `hump_tower` use case packages) **embed** `database.Database` and override only what the test reads,
so a new method compiles everywhere and panics on a nil embedded interface the first time a
test reaches it.

A use case may hold a **nil** `Database`: `get_snapshot_estimates` is constructed without one
when the API is only described, never served, and answers a closed channel.

## `ProcessMessageHandler` is the single entry into the pipeline

WAL rows (`Listen`) and snapshot rows (`adapter/snapshot`) reach the processing use case through
this one signature. `MessageMetadata` follows `headers`, before `prev` and `next`, and carries
`MessageID` and `SnapshotID`. `SnapshotID` is empty for WAL and is the exported database snapshot
id for snapshot rows. A snapshot row is an `insert` with nil `prev` and no headers.

The handler is not safe for concurrent calls — `StartSnapshot` pauses WAL delivery precisely so
the snapshot runner is its only caller.

## The snapshot methods are a protocol, not four calls

`ValidateWhere` → `StartSnapshot` → `ReadSnapshot` (many, concurrently) → `FinishSnapshot`.

- `StartSnapshot` is idempotent while a snapshot is active: it returns the **same id**. A caller
  that restarts must wait for the previous `FinishSnapshot` first, or it gets the dying
  snapshot's id — `adapter/snapshot` does exactly that through `runDone`.
- `FinishSnapshot` is a no-op without an active snapshot, so it is safe in a `defer`.
- Until `FinishSnapshot` runs, WAL delivery stays paused and the slot keeps retaining WAL.

`ReadSnapshot` calls `func(domain.DbRecord, database.SnapshotRowMetadata) error` per row.
`SnapshotRowMetadata.Schema` and `.Table` identify the physical leaf; `.CTID` identifies the
tuple within it. These are separate from user fields in `DbRecord`, including columns named
`ctid`, and from the configured table reported to sinks.

## `GetRowCount` streams, and errors travel inside it

The returned error covers what is known before the counting starts: an invalid filter
(`ErrInvalidWhere`), a failed connection, a failed table-existence check, or a missing table
(`ErrTableNotFound`). After that the channel carries a
rough planner estimate, then the exact count, and closes. A failed estimate arrives as a
`RowCount` with `Error` set, not as a second return value, and ends the stream; a failed exact
count is **dropped silently**, so the rough element may be the only one. A consumer has to read
until the channel closes, prefer the non-`Rough` element, and cope with one or two elements: a lone error element, a
lone rough one, or rough then exact.

## Errors

The sentinels are wrapped by `adapter/postgres` only. **No production code matches them with
`errors.Is`** — they classify the log line and the error text, which `get_snapshot_estimates`
forwards to the client as is. The one that decides an HTTP status is `ErrInvalidWhere`, and only
indirectly, through `port/snapshot.ErrInvalidWhere`; see `port/snapshot/AGENTS.md`.

## `TableType` is a wire format

`adapter/postgres/tables.go` casts the catalog query's output straight into `TableType`, and
`get_snapshot_tables/dto.go` repeats the set as `enum:"regular,partitioned,hypertable"`. Adding
or renaming a value means editing the SQL, the constants and that tag together.
