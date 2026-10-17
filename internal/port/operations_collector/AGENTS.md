# port/operations_collector — how this package hangs together

The dashboard's operations log as its consumers see it: `Record` and `Attempt` going in, `Query`
and `Page` coming out, and the `Collector` between them. One file, `operations_collector.go`.
The ring, the buffer swap and the filter are in `adapter/operations_collector/AGENTS.md`.

The directory is `operations_collector`, the package is **`operationscollector`**, so every
import needs an alias.

## Two implementers, one of them a decorator

`adapter/operations_collector.Collector` is the log. `adapter/live_streamer.Streamer` implements
the same interface by delegating every method to the collector it wraps, and that wrapper is
what `cmd/hump/main.go` hands out. A method added to `Collector` has to be added to the
decorator as well, or the wiring stops compiling.

## Who calls what

- **`process_operation`** needs `Connected` and `Record` only, and declares its own
  `OperationLog` interface for them rather than taking the `Collector`.
- **`get_operations`** calls `Search`; **`stream_updates`** calls `Connect` / `Disconnect`,
  which are a refcount and must be paired. `Dump` has no caller in the application layer —
  `Dump(start, end)` is `Search(Query{Start: start, End: end})`, a window with no filters.
- **Nil is a valid `Collector`.** It is wired only when the web UI is enabled; every consumer
  nil-checks it.

## `Record` transfers ownership

`Record(rec *Record)` takes over `rec.Payload`, `rec.Metadata` and `rec.Attempts` and writes
reusable buffers back into the same fields. This is why the parameter is a pointer, and it is
the part of the contract a caller breaks most easily: it must not keep a reference to what it
passed in, and must continue with what came back. `Dump` and `Search` return deep copies for the
same reason — the slots are overwritten as soon as recording resumes.

`Connected` is a hint checked on every delivery, so it must stay a lock-free read, and `Record`
on an idle collector is a silent no-op rather than an error.

## Vocabulary that is easy to get wrong

- **A retry is an `Attempt`, never a `Record`.** `Record.Status` is `success` or `failure`
  only; `domain.DeliveryStatusRetry` appears on attempts. `get_operations/dto.go` repeats that
  as `enum:"success,failure"` for the status filter.
- **`Target` vs `TargetKey`.** `Target` is what the sink rendered for this row; `TargetKey` is
  the configured connection's target, the one the metrics topology is keyed by. `Query.Targets`
  matches `TargetKey`.
- **`Timestamp` vs `StartedAt`.** The WAL commit time of the row, against when the first
  delivery attempt began.
- **`Payload` is truncated, `PayloadSize` is not.** `Query.Text` only searches what was kept.
- **`Seq` vs `Page.Written`.** `Seq` orders records across the whole process and is assigned by
  the collector, not the caller; `Written` restarts with each activation.
- **Offsets count back from the newest.** `Start`/`End` — in `Dump`, `Query` and `Page` — are
  indices into the *matching* records, half-open, clamped to `Total`. Out of range is an empty
  page, never an error; neither `Dump` nor `Search` returns one.

`SinkType`, `Host` and the `SinkTypes`, `Hosts`, `Targets` and `Tables` filters are plain strings:
this package imports `domain` only, not `port/sink`. `Operations` and `Statuses` are
`[]domain.OperationType` and `[]domain.DeliveryStatus`.
