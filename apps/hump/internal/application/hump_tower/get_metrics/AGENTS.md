# application/hump_tower/get_metrics — how this package hangs together

`POST /api/metrics` (`metrics.get`, registered in `internal/controller/hump_tower.go`). One call
returns the dashboard's whole metrics picture for a bucket width: the bucketed series, lifetime
counts per table and per sink, and the table-to-target edges. All of the aggregation lives in
`adapter/metrics_collector`; this package picks the width, asks for a `Dump` and renders it.

## Layout

| File | Role |
|---|---|
| `usecase.go` | `Metrics(duration)`: nil check, width lookup, `collector.Dump` |
| `controller.go` | the huma handler, the error-to-status mapping, and one renderer per response section |
| `dto.go` | `MetricsRequest` / `MetricsResponse` — the wire contract |

## What it does not own

Read `adapter/metrics_collector/AGENTS.md`, "Two clocks in one response", before changing a field:
`points` are windowed deltas, `tables` / `sinks` / `connections` are lifetime counts, and every
`p95` is windowed and already in **milliseconds** when it gets here. The controller converts
nothing about latency — it only rounds counts.

`toCount` exists because counters arrive as `float64` deltas that are whole numbers; `math.Round`
rather than a truncating cast keeps a `41.999999` from becoming 41.

## The duration is declared twice

The request's `enum:"1m,5m,10m"` tag and `metricscollector.Durations` are two lists of the same
values. huma rejects anything outside the tag with 422 before the handler runs, so
`ErrUnknownDuration` from `Duration.Duration()` is only reachable by calling the use case directly
— but add a width to one list and not the other and it is either unreachable or undocumented.

`duration` is the **bucket width**, not a look-back range: the collector returns as many buckets of
that width as it retains.

## Errors

| Cause | Status |
|---|---|
| `ErrDisabled` (nil collector) | 503 |
| anything else — an unknown duration, a failed `Dump` | 422 |

`UseCase.collector` is nil when the web UI is disabled. `cmd/hump/main.go` builds the collector
whenever the web port is set, and the web UI is only served then, so a served route never sees
nil, whatever `--telemetry-port` is. The nil path is
real for `cmd/openapi`, which constructs the use case with nil purely to register the route, and
for tests. Keep `NewUseCase(nil)` safe: `moon run hump-tower:openapi` depends on it.

## The response shape is the contract

`dto.go` tags are emitted into `apps/hump-tower/src/shared/api/openapi.json` by
`moon run hump-tower:openapi` and become the generated client's types.

- `MetricsTableOps` and `MetricsTableOp` abbreviate their keys (`i`/`u`/`d`/`t`, `n`) because the
  map that carries them repeats them once per configured table. Keep new per-entry keys short for
  the same reason.
- `p95` and `success` are `*float64` with `omitempty`: absent means "no estimate", which is not
  the same as 0.
- `tables` is keyed by `"schema.name"`, and `connections[].table` is the same string, built in
  `connections`. Maps and slices are always allocated, never null.

## Tests

`usecase_test.go` only: the four outcomes of `Metrics`. The renderers have no tests of their own.
`fakeCollector` implements the whole port because `Dump` is the only method reachable from here;
the rest are stubs. Helpers live below the last test function — see the `go` skill.
