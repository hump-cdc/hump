# Propagating trace IDs through Postgres logical replication

Pattern for carrying distributed-tracing context (trace ID, span ID, baggage) from the application that writes to Postgres into the CDC stream, so downstream sinks can correlate change events with the originating request.

## Goal

Each WAL change event delivered to a sink should be tagged with the trace context of the database transaction that produced it, without storing the trace ID in business tables.

## Mechanism

`pg_logical_emit_message(transactional, prefix, content)` writes an arbitrary record into WAL. Logical-replication consumers receive it as `*pglogrepl.LogicalDecodingMessageV2` (with `proto_version '2'` + `messages 'true'` plugin args).

The application emits a **transactional** message at the start of each traced transaction; the consumer buffers it per-xid and attaches the context to subsequent change events of the same transaction.

## Publisher side (application)

Before the first DML of a traced transaction, emit a marker:

```sql
BEGIN;
SELECT pg_logical_emit_message(
    true,            -- transactional: discarded on rollback, ordered with data
    'trace',         -- prefix used by the consumer to dispatch
    '{"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7"}'
);
INSERT INTO orders ...;
INSERT INTO order_items ...;
COMMIT;
```

Recommended implementation: hook into the database driver / connection pool so any transaction that has an active OpenTelemetry span automatically emits the marker on its first statement. Avoids relying on every call site remembering to do it manually.

`prefix` is a free-form string; pick a stable name (e.g. `"trace"` or `"otel"`) and treat it as your contract with consumers.

`content` can be `text` or `bytea`. Recommended: JSON with a tiny stable schema (trace ID, span ID, optional baggage). Keep it small — every traced transaction adds this to WAL.

## Consumer side (this project)

Required `pluginArguments` in `apps/hump/internal/adapter/postgres/wal.go`:

```go
pluginArguments: []string{
    "proto_version '2'",
    fmt.Sprintf("publication_names '%s'", publicationName),
    "messages 'true'",   // re-enable
},
```

Per-xid buffer and dispatch in `processMessage`:

```go
case *pglogrepl.LogicalDecodingMessageV2:
    if !message.Transactional {
        return nil   // out-of-band marker; ignore for tracing
    }
    if message.Prefix != "trace" {
        return nil   // not ours
    }
    var tc TraceContext
    if err := json.Unmarshal(message.Content, &tc); err != nil {
        return fmt.Errorf("decode trace message xid=%d: %w", message.Xid, err)
    }
    w.pendingTrace[message.Xid] = tc

case *pglogrepl.InsertMessageV2:
    // ...
    tc := w.pendingTrace[message.Xid]   // zero value if no trace context
    // pass tc into w.process(...)

case *pglogrepl.CommitMessageV2:
    delete(w.pendingTrace, message.Xid)   // also delete on rollback if streaming
```

`pendingTrace` is `map[uint32]TraceContext`. Add it to the `Wal` struct and initialise alongside `relations`/`typeMap` in `Start`.

`ProcessHandler` needs to grow a way to carry the trace context — either a new explicit parameter or a metadata map alongside `prev`/`next`. Sinks then attach the trace ID to outgoing payloads (Kafka headers, HTTP headers, JSON envelope, etc.).

## Why transactional, not non-transactional

| | transactional = true | transactional = false |
|---|---|---|
| Bound to surrounding txn | yes | no |
| Streamed if txn commits | yes | yes |
| Streamed if txn rolls back | **no** | **yes** |
| Ordered with data | yes (between Begin and Commit) | no |

For trace propagation you want `true`:
- A rolled-back transaction shouldn't produce a trace span for non-existent operations.
- The marker arrives **before** the data records of the same transaction, so the consumer can stamp every record without buffering.

## Caveats

1. **Application discipline.** Every transaction that should be traced must emit the marker before its first DML. Records from un-marked transactions will have empty trace context — sinks must handle that gracefully (e.g. emit without trace headers rather than crash).

2. **No PII in the message.** The CDC stream is typically delivered to systems with weaker access controls than the source DB. Trace IDs and span IDs are safe; user IDs / emails / tokens are not.

3. **Size.** `pg_logical_emit_message` accepts up to ~1 GB per call. Keep payloads to a few hundred bytes — a tight JSON object with trace ID, span ID, and minimal baggage.

4. **WAL cost.** One extra WAL record per traced transaction. Negligible compared to the data writes, but it does count against `max_slot_wal_keep_size`.

5. **Concurrent transactions.** With `proto_version '2'` and streaming **off** (current setup), pgoutput delivers committed transactions one at a time in commit order — `pendingTrace` is keyed by `Xid` which keeps things correct even if Postgres interleaves during decoding. If you ever enable `streaming 'true'`, the same xid keying still works, but you must also handle `StreamAbortMessageV2` to drop the buffered trace context for aborted streamed transactions.

6. **REPLICA IDENTITY FULL is unrelated.** Trace IDs ride on logical messages, not on tuple data, so replica identity does not affect them. Required for the rest of the project's TOAST handling (`EnsureReplicaIdentityFull` in `apps/hump/internal/adapter/postgres/bootstrapper.go`).

7. **Memory.** `pendingTrace` is bounded by concurrent in-flight transactions on the publisher (typically small). Always `delete(w.pendingTrace, xid)` on `CommitMessageV2` (and `StreamAbortMessageV2` if streaming) so the map doesn't leak entries from never-emitted-trace transactions. To be safe against application bugs, periodically purge entries older than some max-transaction-age threshold.

## Required cases in `processMessage`

When implementing, add explicit cases for messages your switch doesn't currently handle:

- `*pglogrepl.LogicalDecodingMessageV2` — described above.
- `*pglogrepl.BeginMessageV2` — optional; useful only if you want to capture `Xid` early for logging.
- `*pglogrepl.CommitMessageV2` — required, to free the `pendingTrace` entry.

Without the Commit case, the buffer leaks per-transaction memory until process restart.

## Plugin argument summary

Final `pluginArguments` for trace propagation:

```go
[]string{
    "proto_version '2'",
    fmt.Sprintf("publication_names '%s'", publicationName),
    "messages 'true'",
}
```

Do not enable `streaming 'true'` unless you also add `StreamCommitMessageV2` / `StreamAbortMessageV2` handling — current code does not.

## References

- Postgres docs: `pg_logical_emit_message` — https://www.postgresql.org/docs/current/functions-admin.html#FUNCTIONS-ADMIN-LOGICAL-FUNCS [unconfirmed link, verify]
- pgoutput protocol: https://www.postgresql.org/docs/current/protocol-logicalrep-message-formats.html
- pglogrepl message types: `*pglogrepl.LogicalDecodingMessageV2` in `github.com/jackc/pglogrepl`
