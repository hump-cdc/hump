# application/processing/process_operation — how this package hangs together

The middle of the pipeline. One row change comes in as a flat argument list, becomes one
`port/sink.Context`, and is delivered to every sink configured for its table, in parallel,
with retries, a per-attempt timeout, a failure policy, metrics and — when somebody is watching —
one operations log entry per sink. `Process` returns only after every sink has finished, which is
what lets the caller advance its position on a nil.

## Layout

| File | Role |
|---|---|
| `usecase.go` | `UseCase`: builds the bindings and the worker pool, `Process` fans one row out |
| `binding.go` | `binding` — one (table, sink) pair, its delivery chain, hooks and scratch record |
| `controller.go` | `Controller`, a pass-through whose `Process` matches `database.ProcessMessageHandler` |
| `logger.go` | `log.Scope("application/process_operation")` |

## Two callers, one use case, one goroutine at a time

`cmd/hump/main.go` builds a single `UseCase` and hands `Controller.Process` to both producers:

- **WAL** — `internal/controller/process.go` passes it to `db.Listen`. The context it receives is
  detached (`context.WithoutCancel` in `adapter/postgres/wal.go`), so a shutdown signal does not
  cancel a delivery; `shutdown.Draining()` is what ends its retries.
- **Snapshot** — `adapter/snapshot` receives it as a func value and calls it once per row with
  `insert`, `prev` nil and empty headers, on the run's own cancellable context.

`pkg/worker_pool`'s batch API is single-goroutine only. `Process` returns the error of
`BeginBatch` wrapped as `begin batch: …`, so a second concurrent caller fails with
`ErrAlreadyInFlight` and delivers nothing; it is not queued behind the first. Keeping callers from
overlapping is still the job of the code outside: the snapshot pauses WAL delivery (`walGate`), and
serializes its own table readers with `processMu`. Any new caller has to take part in that.

A call after shutdown fails the same way: `BeginBatch` returns `ErrWorkerPoolClosed`, and `Process`
returns it wrapped without submitting anything.

## Bindings and the pool

`NewUseCase` walks `cfg.ResolveAll()` and builds one `binding` per sink of every table, keyed by
`domain.TableIdentifier`; a table listed twice accumulates the sinks of both entries. Everything
that can be resolved once is resolved here — the Prometheus children, the operation set, the whole
delivery chain — so `Process` only looks the slice up, filters it through `accepts`, and submits.
A table with no bindings, or a row whose operation no sink accepts, is an empty batch and a nil
return, not an error.

`operations` defaults to all four types in `config/sink`, so `accepts` filters only what a sink
explicitly opted out of.

The pool is sized to the widest table: workers = queue size = the largest number of sinks on one
table, with `BackpressureBlock`. One batch therefore never waits for a free worker, and sinks of
one row never run behind each other.

Errors take two routes, and they do not carry the same information:

- `FlushBatch` returns the bare `ErrWorkerFailed` — the caller learns *that* a sink failed, not
  which one or why;
- the actual error goes to `wp.Errors()` (buffer 1024, overflow dropped and counted), drained by a
  goroutine started in `NewUseCase` into `logger.Error("Worker error")`.

A `Submit` error wins over the batch result, but the batch is still flushed first, so tasks
already queued are awaited either way.

## The delivery chain

`newBinding` composes the chain once. Outermost first:

```
WithFailureMode → WithStatus → withOperation ─┬─ plain:  WithRetry → WithTimeout → sink.Process(capture = nil)
                                              └─ logged: WithRetry → withAttempt → WithTimeout → sink.Process(&op.capture)
```

The order is the design, and each position has a reason:

- **`WithTimeout` innermost** — the timeout is per attempt (`retry.attemptTimeout`, 10s by
  default), not per delivery. Its `context.WithTimeout` is the binding's only source of
  allocations, four per attempt at the time of writing (`BenchmarkBindingProcessWithoutLog`).
- **`withAttempt` around the timeout** — so a deadline is recorded as that attempt's error and the
  attempt's full duration is measured.
- **`WithRetry` around both** — up to `retry.maxAttempts` (5 by default), exponential backoff
  1s → 30s with full jitter. It retries on **any** non-nil error; `sink.ErrSinkPermanent` is not
  consulted (see `adapter/sink/AGENTS.md`). Its stop channel is `shutdown.Draining()`: a drain that
  starts mid-backoff cuts the pause short for one immediate last attempt, and a failure after the
  drain has started ends the loop with `ErrRetryAborted` wrapping the attempt's error — if another
  attempt remains. A failure on the last attempt comes back bare, so `failure: skip` swallows it.
- **`withOperation` outside the retry loop** — so it sees the terminal error and writes one record
  per delivery rather than one per attempt.
- **`WithStatus` outside `withOperation`** — so the counters and the log entry describe the same
  outcome.
- **`WithFailureMode` outermost** — so a delivery dropped by `failure: skip` is still counted and
  logged as a `failure` before its error is swallowed. In skip mode a cancelled context and
  `ErrRetryAborted` are still returned: that row never had its full retry budget.

With the default `failure: crash` the error reaches the caller. For WAL that ends `Listen` (the
error is joined with `database.ErrWal`), which stops the runner; the position was not advanced
past the transaction, so the row is delivered again on restart. Delivery is at-least-once, and a
sink that succeeded for that row before a sibling failed sees it twice.

## The binding is its own hooks

`binding` implements `resilience.RetryHooks`, `StatusHooks` and `SkipHooks` for its own chain:

| Hook | When | What it does |
|---|---|---|
| `OnDuration` | after every attempt | `sink_duration_seconds` — per attempt, not per delivery |
| `OnRetry` | after the backoff pause, before the next attempt | `sink_messages_total{status="retry"}` + observer |
| `OnSuccess` / `OnFailure` | once per delivery | `sink_messages_total` + observer |
| `OnSkip` | a failure swallowed by skip mode | an error log line with sink and table |

Both metrics are declared without a prefix; the factory from `pkg/observability/metric` adds `hump_`,
and `db` is a constant label. The `host` and `target` labels come from `cfg.Sink.Metadata` rendered **once, at construction**,
from the schema and table alone. They are the configured target, not what a templated topic or URL
renders to per row, and they have to match what the dashboard's topology computes —
`adapter/metrics_collector/AGENTS.md` explains why byte for byte.

## `Observer` and `OperationLog` are declared here

Both interfaces live in `usecase.go`, each the slice of a port this package actually calls, and
both are nil-able:

- `Observer` — `metrics_collector.Collector.Record`. The sink metrics carry no table label, so the
  dashboard's per-connection counts are pushed through this instead.
- `OperationLog` — `Connected` plus `Record`, satisfied by `adapter/live_streamer` wrapping
  `adapter/operations_collector`.

`main.go` constructs both only when the web UI port is set, and passes them as port-typed
variables that stay nil otherwise. A nil interface converted to another interface stays nil, which
is what `b.observer == nil` and `b.log == nil` rely on. `TestBindingSkipsRecordWithoutObserver`
pins it for `observer` only; no test covers `log`. Never hand in a typed nil pointer.

## The operations log path

Two chains rather than one with a flag: the logged chain writes into the binding's scratch state
from every attempt, so the choice is made once per delivery, in `withOperation`, outside the retry
loop. It takes `plain` when the log is nil, when `Connected()` is false, or when the scratch record
is already in use.

`Connected()` is a hint, not a guarantee — the last client can leave before `Record` runs, and the
collector then drops the record silently. Nothing here depends on it being stored.

`operation` is that scratch state: one `operationscollector.Record` and one `sink.Capture` per
binding, reused so a logged delivery allocates nothing of its own.

- `inflight` guards the reuse. The pool submits a binding once per awaited batch, so two deliveries
  never overlap; the CAS exists for the day that breaks, and such a delivery is **delivered but not
  logged** rather than corrupting the record.
- `begin` resets the capture and the record, keeping every buffer.
- `end` copies the capture in, sets the terminal status and error, and labels the attempts. An
  attempt's status is only knowable once the loop is over: a failed one is `retry` unless it is the
  last attempt of a failed delivery. `Retries` is attempts minus one.
- `Record` **swaps buffers** with the collector: it keeps `Payload`, `Metadata` and `Attempts` and
  hands back the ones it held before. `rebind` re-points the capture's `Payload` and `Metadata` at
  what came back. Skip it and, with the real collector, the next delivery appends into the record
  the ring already stored. No test catches a missing rebind: `fakeOperationLog` deep-copies
  `Payload`, `Attempts` and `Metadata` before storing them, so
  `TestBindingRebindsCaptureAfterRecord` still passes without the call.

`Target` starts as the configured target and is overwritten by `capture.Target` when the sink
reports the rendered one; `TargetKey` always stays the configured one, because that is what the
flows inspector filters by.

The ring, the swap's allocation contract and the paging live in
`adapter/operations_collector/AGENTS.md`; what a sink writes into the capture, and when, in
`adapter/sink/AGENTS.md`.

## Shutdown

`NewUseCase` registers `wp.Shutdown` at `shutdown.StageWorkers`. By then the runner has already
called `shutdown.Drain()` and waited (30s at most) for `Listen` to return, so retries have stopped
and normally no batch is open. `Shutdown` takes the pool mutex that `BeginBatch` holds until
`FlushBatch`, so a batch still open is waited for — within the 5s every hook gets.

## Tests

`binding_test.go` covers the binding in isolation: the nil observer, one observer call per outcome,
and `accepts`. `operation_log_test.go` drives a whole chain through `newBinding` with a scripted
sink: no record and no capture while disconnected, one record per delivery whatever the retries,
the attempt status matrix, the capture filled on a failed delivery, `TargetKey` against the
rendered `Target`, the terminal error, two deliveries in a row (which does not pin the rebind, see
above), buffer reuse across a hundred deliveries, and
the counters still firing while logging.

Nothing tests `UseCase.Process` — the fan-out, the pool sizing and the error precedence are
unpinned.

The benchmarks are a contract with `adapter/operations_collector`, which cites them by name:
`BenchmarkBindingProcessWithoutLog` (nil log) is the control for `BenchmarkBindingProcessIdle`
(log present, nobody connected), and the two must report the same allocation count — that is the
proof a disconnected log costs nothing. `BenchmarkBindingProcessLogged` is the connected path and
is not allocation free by design.

`newTestBinding` builds a binding over a private registry with a `fakeSink` that fails the first
`len(errs)` attempts; `newTestContext` is one row by id; `deliver` runs row `"a"` and tolerates
`errBoom`. `fakeOperationLog` copies what it is handed, because the real collector swaps the
buffers away, and with `recycle` hands buffers back like a ring with populated slots;
`benchOperationLog` swaps and keeps nothing. Helpers live below the last test function — see the
`go` skill — except `fakeObserver`, which sits in `binding_test.go` right after
`TestBindingRecordsEveryOutcome`. `TestBindingKeepsCountersWhileLogging` in `operation_log_test.go`
uses it too.
