# application/hump_tower/get_metrics — how this package hangs together

`POST /api/metrics` (`metrics.get`). One call returns the dashboard's whole metrics picture for a
bucket width: the bucketed series, lifetime counts per table and per sink, and the table-to-target
edges. All of the aggregation lives in `adapter/metrics_collector`; this package picks the width,
asks for a `Dump` and renders it.

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

## Nil collector

A nil collector is `ErrDisabled` → 503; any other error is 422. `cmd/hump/main.go` builds the
collector whenever the web port is set, so a served route never sees nil, whatever
`--telemetry-port` is.

## The response shape is the contract

- `MetricsTableOps` and `MetricsTableOp` abbreviate their keys (`i`/`u`/`d`/`t`, `n`) because the
  map that carries them repeats them once per configured table. Keep new per-entry keys short for
  the same reason.
- `p95` and `success` are `*float64` with `omitempty`: absent means "no estimate", which is not
  the same as 0.
- `tables` is keyed by `"schema.name"`, and `connections[].table` is the same string, built in
  `connections`. Maps and slices are always allocated, never null.

## Tests

`usecase_test.go` covers the four outcomes of `Metrics`. The renderers have no tests of their own.
