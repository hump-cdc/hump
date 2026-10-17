# port/metrics_collector — how this package hangs together

The dashboard's metrics source as its consumers see it: the `Collector` interface, the bucket
widths it supports, and the `Snapshot` one `Dump` returns. `adapter/metrics_collector` is the
only implementation — read `adapter/metrics_collector/AGENTS.md` before changing what a field
means, because the semantics are decided there.

The directory is `metrics_collector`, the package is **`metricscollector`**, so every import
needs an alias.

## Layout

| File | Role |
|---|---|
| `metrics_collector.go` | `Duration`, the `Snapshot` tree, `Connection`, the `Collector` interface |
| `errors.go` | `ErrUnknownDuration` |
| `metrics_collector_test.go` | `Duration.Duration()` — the only test in any port package |

## `Duration` is a wire format and a key set

- The literals are the API contract: `get_metrics/dto.go` repeats them as `enum:"1m,5m,10m"`,
  and nothing derives that tag from the constants.
- `Durations` is what the adapter allocates its series from — one per width, at activation.
- `Duration.Duration()` is the single place a literal becomes a `time.Duration`.

A new width therefore touches four places: the constant, `Durations`, the `switch` in
`Duration()`, and the DTO enum. `ErrUnknownDuration` reaches the client through
`get_metrics`, whose controller answers 422 for any error that is not its own `ErrDisabled`.

## Two clocks in one `Snapshot`

`Points` are windowed deltas of the requested width. `Tables`, `Sinks` and `Connections` are
lifetime counts since the process started and do not depend on the duration — except every
`P95`, which is windowed and in **milliseconds**. The pointer fields (`P95`, `Success`) are nil
when there was nothing to compute from, which is not the same as zero; keep them pointers.

`Sinks` and `Connections` are keyed by the configured topology, so an entry with no traffic is
present with zeros.

## `Connection` strings must match the metric labels

`Host` and `Target` have to equal, byte for byte, what `Definition.Metadata(SinkContext{Schema,
Table})` renders — `cmd/hump/main.go` `topology`, `process_operation.newBinding`,
`get_config/usecase.go`, `get_snapshot_tables/usecase.go` and `adapter/snapshot/snapshot.go`
all make that call. `SinkType` is a
plain `string` here, not `port/sink.SinkType`: this package does not
import `port/sink`, and should not start to.

## Who calls what

- **`process_operation`** needs only `Record`, and declares its own one-method `Observer`
  interface for it rather than taking the `Collector`. `Record` is called per delivery on every
  worker goroutine and must stay safe while the collector is idle.
- **`get_metrics`** calls `Dump`; **`stream_updates`** calls `Connect` / `Disconnect`;
  **`adapter/live_streamer`** calls `Scrape` to hold a window back until the counts catch up.
- **Nil is a valid `Collector`.** It is wired only when the web UI is enabled; every consumer
  nil-checks it.

`Connect` and `Disconnect` are a refcount, so they must be paired by the caller.
