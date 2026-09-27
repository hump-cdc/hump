# adapter/postgres — how this package hangs together

CDC source adapter. Two jobs: bootstrap the server-side replication state to match the
configuration, then decode the logical replication stream and hand each row change to the
application layer keyed by the *configured* table.

## Layout

| File | Role |
|---|---|
| `postgres.go` | entry points — `Listen`, `Cleanup`, and `configure`, which orders the whole bootstrap |
| `bootstrapper.go` | server-side DDL: publication, replication slot, `REPLICA IDENTITY FULL` |
| `introspector.go` | catalog reads; resolves an incoming relation to the configured table that owns it |
| `introspector_registry.go` | in-memory config: configured tables plus the partition→parent index |
| `wal.go` | pgoutput stream: relation cache, row decoding, standby keepalive |
| `replication_lag.go` | polls `pg_replication_slots` on its own session for the `replication_lag_bytes` gauge |
| `connection.go` | self-healing `pgconn` session with retry/backoff |
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

## Stream mechanics worth knowing

- pgoutput runs with `proto_version '2'`, streaming **off**, so only committed transactions
  arrive, in commit order. A chunk's catalog rows are therefore visible by the time the INSERT
  that created it is decoded — which is what makes the parent lookup safe.
- A registry miss can block on the database for a long time (catalog round trips, reconnects,
  an `ALTER` waiting on a lock), so `wal.resolveTable` starts a standby-keepalive goroutine for
  the duration; otherwise the upstream walsender times the session out.
- See `docs/TRACE_ID.md` before enabling streaming — `pendingTrace` is keyed by xid and would
  need `StreamAbortMessageV2` handling.
