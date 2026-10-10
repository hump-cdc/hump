# adapter/postgres — how this package hangs together

CDC source adapter. Two jobs: bootstrap the server-side replication state to match the
configuration, then decode the logical replication stream and hand each row change to the
application layer keyed by the *configured* table.

## Layout

| File | Role |
|---|---|
| `postgres.go` | `New` and `configure`, which orders the whole bootstrap |
| `postgres_streaming.go` | `Listen`: bootstrap, then the lag poller and the WAL stream, each on a session of its own |
| `postgres_cleanup.go` | `Cleanup`: drops the publication and the replication slot |
| `postgres_snapshot.go` | `ValidateWhere`, `StartSnapshot`, `FinishSnapshot`, `ReadSnapshot`: one-line entry points into `snapshot` |
| `postgres_metadata.go` | getters for what `New` was given: database name, host and port, publication, slot |
| `bootstrapper.go` | server-side DDL: publication, replication slot, `REPLICA IDENTITY FULL` |
| `introspector.go` | catalog reads; resolves an incoming relation to the configured table that owns it |
| `introspector_registry.go` | in-memory config: configured tables plus the partition→parent index |
| `wal.go` | pgoutput stream: relation cache, row decoding, standby keepalive |
| `replication_lag.go` | polls `pg_replication_slots` on its own session for the `replication_lag_bytes` gauge |
| `connection.go` | self-healing `pgconn` session with retry/backoff |
| `connection_config.go` | `connectionOptions`: connection string parsing, the replication flag, the retry policy and its options |
| `table.go` | `tableType` / `tableMetadata`, the param builders (`tableArrayParams`, `tableScalarParams`, `textParam`) and `quoteIdentifier` / `quoteLiteral` / `qualifiedTableName` |
| `table_metadata.go` | `tableMetadataLoader`: the TimescaleDB probe and the catalog query for tables with their partitions |
| `metrics.go` | the `replication_lag_bytes` gauge and the `processed_rows_duration_seconds` histogram, with its cached per-table observers |
| `errors.go` | `ErrTableIgnored` and the package-internal sentinels |
| `postgres_row_count.go` | `GetRowCount`: planner estimate then exact `count(*)`, each in a `READ ONLY` transaction under `statement_timeout`, on a session of its own per call |
| `tables.go` | `GetTables`: one catalog query per call, on its own session, describing each requested table's type and columns; tables missing from the database are skipped |
| `snapshot_where_suggester.go` | `snapshotWhereSuggester`: picks the temporal column most likely to track recency (time key → updated → created → event-time names → sole `now()` default) and suggests the last 7 days of it |
| `snapshot_where_parser.go` | lexer + recursive-descent parser for the `GetRowCount`/snapshot filter; re-renders it with every identifier and literal quoted, so the raw string never reaches the server |
| `wal_gate.go` | `walGate`: pauses delivery to the WAL handler while a snapshot is active |
| `snapshot.go` | `snapshot`, built once in `New`: `ValidateWhere`; `Start` / `Finish` and the holder session that keeps an exported snapshot alive; `Read`: leaves + columns inside the snapshot, then `COPY` chunks by `ctid` page range |
| `snapshot_export_parser.go` | `snapshotExportParser`: parser for the `COPY … TO STDOUT` text format, decoding fields with the same `decodeText` WAL uses |
| `sql/` | every statement, one file each, pulled in with `//go:embed` |

## Conventions

- **SQL lives in `sql/*.sql`, never inline.** Parameterized statements go through
  `connection.ExecParams` with `[]param`; use `tableArrayParams` for the `(schemas[], names[])`
  shape most catalog queries filter on, `tableScalarParams` / `textParam` for scalars.
  Statements built by string formatting must quote through `quoteIdentifier` / `quoteLiteral`.
- **Catalog queries come in two flavours**, TimescaleDB and not, picked by a probe that is
  cached per session (`tableMetadataLoader.hasTimescaleDB`) — the extension cannot appear or vanish
  mid-session.
- **Tests talk to a real socket** against `fakeServer` (`postgres_test.go`), whose responders
  dispatch on substrings of the SQL. A new query therefore needs a distinctive marker in its
  text — the leading `--` comment is the intended hook — and a case in the shared responders.
  Test helpers live below the last test function; see the `go` skill.

## The TimescaleDB constraint everything else follows from

PostgreSQL has no way to publish "this table and its future inheritance children", and
TimescaleDB does not add new chunks to a publication on its own
([#8628](https://github.com/timescale/timescaledb/issues/8628)). So for a configured
hypertable, `EnsurePublicationExists` emits `TABLES IN SCHEMA "_timescaledb_internal"` — the
same workaround ClickHouse ClickPipes documents. Two consequences, and both are load-bearing:

**1. Foreign relations reach the stream.** Chunks of hypertables hump does not replicate,
compressed chunks, continuous aggregate materialization tables — all of them arrive as
relations. `publish_via_partition_root` does not fold them into the parent: it applies to
declarative partitioning only, and TimescaleDB chunks are inheritance children of a
`relkind = 'r'` table. So the chunk→parent mapping is done in Go, and `decodeRow` reads the
*chunk's* columns while the reported `ref` is the parent.

The miss path is therefore hot and must stay cheap. `ResolveTableConfiguration` goes:
registry hit → **one narrow `pg_inherits` lookup** of the immediate parent
(`sql/introspector_table_parent.sql`) → full `invalidate` **only** when a configured table
owns the relation → otherwise straight into `ignoredTables`. Do not put a query that scales
with the number of chunks back on that path — the metadata query returns one row per chunk of
every configured hypertable, and a foreign hypertable mints a new relation every chunk
interval. One `pg_inherits` query serves chunks and declarative partitions alike, so this path
needs no TimescaleDB branch.

Known and accepted: `ignoredTables` and `wal.relations` are never evicted.

**2. Publishing a schema breaks writes hump has nothing to do with.** A table in a publication
that publishes UPDATE/DELETE must have a usable replica identity, or the server rejects those
statements. Reproduced on the stand: with the publication up, `UPDATE` on a PK-less hypertable
outside the configuration and `refresh_continuous_aggregate` both fail. No client-side change
fixes this, so `WarnMissingReplicaIdentity` runs as a preflight after the publication is set
and names the affected tables in the log. It is diagnostics: a failed lookup is logged and
swallowed, never fatal.

## Replica identity

hump sets `REPLICA IDENTITY FULL` on configured tables marked `replica: full` and on their
declarative partitions. **TimescaleDB chunks are deliberately excluded**
(`introspector.replicaIdentityTargets`): TimescaleDB ≥ 2.12 propagates the hypertable's setting
to existing chunks and copies it onto new ones — verified on 2.26.3 / PG 17.9, both a pre-ALTER
and a post-ALTER chunk came back `relreplident = 'f'`. Compressed chunks (`compress_hyper_*`)
do *not* inherit it, but their DML goes through the table AM rather than the executor, so the
server-side check never fires for them.

`ALTER TABLE … REPLICA IDENTITY` is issued one statement per table, never batched: the simple
protocol would run a batch as one implicit transaction and hold every `AccessExclusiveLock`
until the last one commits.

**What `replica: unmanaged` costs at decode time.** FULL is what makes `decodeRow` able to fill
in an unchanged TOAST column (`'u'`) — `ExtractReplicaIdentity` flattens the old tuple, so it
arrives as `'t'` with the real value. Without FULL the old tuple is either absent or key-only
(`'K'`), and a key-only tuple nulls out every column outside the replica identity. Those nulls
are absences, not values, so `decodeRow` reports them — and any `'u'` it cannot resolve — as
`domain.Unavailable` rather than `nil`. The distinction is invisible in a payload (the sentinel
renders as `""`); only the `unavailable` template func exposes it.

`relation.Columns[i].Flags == 1` means "part of the effective replica identity", **not** "primary
key": under FULL pgoutput sets it on every column. It only discriminates while
`OldTupleType == 'K'`, which is the one place `decodeRow` reads it.

## Snapshots

- **Pause.** `StartSnapshot` closes `walGate` first and waits until no handler call is in flight,
  so the snapshot runner is the only caller of the (non-reentrant) processing use case. The gate
  sits inside `processMessage`, i.e. between `beginProcessing`/`endProcessing`, so the keepalive
  worker keeps the walsender session alive while WAL waits. The wait ends on `Listen`'s own
  context (handlers get a detached one), so a shutdown is not held up by a paused gate. The slot
  keeps WAL for the whole pause. Pause is a flag: repeated `StartSnapshot` returns the same id,
  one `FinishSnapshot` resumes. `pause` has no timeout of its own: a WAL row stuck in sink
  retries keeps `StartSnapshot` waiting until the run is stopped.
- **Holder.** One session runs `BEGIN REPEATABLE READ READ ONLY; LOCK TABLE … IN ACCESS SHARE MODE;
  SELECT pg_export_snapshot()` and stays idle in that transaction (`idle_in_transaction_session_timeout`
  is zeroed for it) until `FinishSnapshot` commits. The `LOCK` waits at most `snapshotLockTimeout`
  (5s, set as `lock_timeout`), so `StartSnapshot` fails rather than queue behind DDL holding those
  tables. The lock recurses to partitions and chunks and
  is what keeps `ctid`s stable: without it `VACUUM FULL`/`CLUSTER`/`TRUNCATE` could rewrite a table
  between chunks. It also holds back DDL on those tables, and the snapshot holds `xmin`.
- **Not aligned with the slot.** The slot is created by `pg_create_logical_replication_slot`, so
  there is no snapshot exported at its LSN. WAL resumed after a snapshot replays changes the
  snapshot already saw; consumers converge, but may briefly see older values. Accepted.
- **Leaves.** `pg_inherits` recursion from the table, `relkind = 'r'` only: a regular table is its
  own leaf, a partitioned table's parent has no storage and drops out, a hypertable keeps its
  (usually empty) parent plus its chunks. Compressed chunks are skipped (TimescaleDB flavour of
  the query) — their rows live in `compress_hyper_*` and are deliberately not read.
- **Columns** come from the parent's `pg_attribute` (generated columns excluded, matching pgoutput)
  and are listed explicitly in the `SELECT`, since a leaf's physical attribute order may differ.
- **Reads.** Every chunk is one simple-protocol message — `BEGIN; SET TRANSACTION SNAPSHOT; COPY
  (SELECT … FROM ONLY leaf WHERE ctid range AND (filter)) TO STDOUT; COMMIT` — on a raw session, so
  a reconnect never silently replays a chunk. `COPY` takes no bind parameters, so the id, ranges
  and filter are rendered through `quoteLiteral` and `snapshotWhereParser`. Chunks are
  `snapshotChunkPages` pages; the last one is open-ended.
- **Metrics.** Each processed snapshot row is observed in `processed_rows_duration_seconds` as an
  `insert` of the requested table, so the dashboard's table counts include snapshot rows. WAL and
  snapshot reads observe concurrently, hence the mutex around the `rowObservers` cache.

## Stream mechanics worth knowing

- pgoutput runs with `proto_version '2'`, streaming **off**, so only committed transactions
  arrive, in commit order. A chunk's catalog rows are therefore visible by the time the INSERT
  that created it is decoded — which is what makes the parent lookup safe.
- **The confirmed position moves on two events only**: a Commit sets it to the transaction's
  `TransactionEndLSN`, and a keepalive *outside* a transaction sets it to `ServerWALEnd`, which
  releases the WAL pgoutput skipped (unpublished tables, internal records). Row messages never
  move it, so a restart re-sends an unfinished transaction whole. A keepalive between Begin and
  Commit is ignored: the sender reads a transaction's commit record before replaying it, so its
  `ServerWALEnd` can already lie past rows not yet processed, and confirming it would make the
  server skip that transaction on restart.
- Processing one message can block for a long time — a registry miss (catalog round trips,
  reconnects, an `ALTER` waiting on a lock), a sink stuck in retries, a paused `walGate` — so
  `Start` runs one `keepAlive` worker for the whole stream. It sends a standby status update
  every `standbyInterval`, but only while the receive loop is inside `processMessage`
  (`beginProcessing`/`endProcessing`), reporting the position the loop held on entry; otherwise
  the upstream walsender times the session out. Outside `processMessage` the receive loop is the
  session's only user, and `endProcessing` waits out a flush the worker has in flight.
- See `docs/TRACE_ID.md` before enabling streaming — the `pendingTrace` map it proposes (not
  implemented) is keyed by xid and would need `StreamAbortMessageV2` handling.
