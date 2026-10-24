# entities/metrics — how this slice hangs together

The FSD `entities` slice that owns the dashboard's one metrics query: `useMetrics(duration)` over
`POST /api/metrics`, and the normalized shape every chart, the flows canvas and the operations
filter read. It has no `ui/`. The backend half is
`apps/hump/internal/application/hump_tower/get_metrics/AGENTS.md`; what the numbers mean is in
`apps/hump/internal/adapter/metrics_collector/AGENTS.md`, "Two clocks in one response".

## Public API

No `index.ts`: every consumer is another entity, so the `@x` files are the whole public API.
`normalizeMetrics`, `refetchInterval` and `METRICS_POINTS_MAX` are exported from `use-metrics.ts`
for the test only.

| File | Consumer | Exposes |
|---|---|---|
| `@x/statistics.ts` | the seven chart slices of `entities/statistics` | `useMetrics` |
| `@x/flows.ts` | `entities/flows/flows` | `useMetrics`, `MetricsTopology`, `MetricsSink` |
| `@x/inspector.ts` | `entities/flows/inspector` | `useMetrics`, `MetricsTopology`, `MetricsTable`, `MetricsDelivery` |
| `@x/operations.ts` | `entities/operations/filter` | `useMetrics` |

The files are named after the consumer, not consistently by group or slice (`statistics` and
`operations` are groups, `flows` and `inspector` are slices). A new consuming entity gets **its
own** `@x/<consumer>.ts` with only what it needs; do not point it at an existing file or widen one
for a different consumer. A consumer above `entities` would need an `index.ts` added first.
`biome.json` does not enforce this: its public-API rule lets any layer import an `@x` file.

Every consumer passes the chart step from `useLiveControlsChartStep` (`entities/config/state`), so
they share one cache entry and one request.

## From response to `Metrics`

The cache holds the **raw response**; `normalizeMetrics` runs in `select`. Reading the cache
directly through `keys.queries.metrics(duration)` gives a `MetricsResponseBodyModel`, not a
`Metrics`.

- **Counters are `bigint` at runtime.** The generated client parses responses with zod, which
  coerces `int64` fields to `bigint`, while the generated types say `number`. `toNumber` /
  `toNumbers` handle that: `bucketSeconds`, `delivered`, `failed`, `sinkDelivery`, every `n`,
  `operations`, and a connection's `success` / `failure`. Floats (`throughput`, the gauges, `p95`,
  the `success` share) pass straight through; `sinkLatencyP95` is a float map that `toPoint` still
  runs through `toNumbers`, a no-op for it. A new integer field needs the conversion even though
  TypeScript will not ask for it.
- **The wire abbreviations stop here.** `i` / `u` / `d` / `t` become `insert` / `update` /
  `delete` / `truncate`, `n` becomes `count`, and the `tables` map becomes an array with the key
  as `name`.
- **Absent stays absent.** `p95` and `success` are copied as `undefined`, never defaulted to 0; a
  sink type without an estimate is missing from `sinkLatencyP95`.
- **Topology is sorted** so a layout does not move between refetches: tables by name, sinks by
  type then host, targets by name, locale-independent. `connections` keep the response order.

`MetricsDuration` (`model/types.ts`) is an alias of the generated request's `duration` field, so a
bucket width added on the backend arrives with the regenerated client. `CHART_STEPS` in
`entities/config/state` is a separate hand-written list, checked against it with `satisfies`.

## The fixed timeline

`points` always has `METRICS_POINTS_MAX` (20) entries, oldest first, `bucketMs` apart. Points are
looked up by exact timestamp, and a slot with no match becomes `emptyPoint`, so a bucket the
collector skipped is a zero bar instead of shifting the bars after it.

- The timeline ends at the **newest point of the response**, not at the current time. Only an
  empty response falls back to `now`, floored to the bucket width.
- A point that is not a whole number of buckets before the newest one is dropped.
- An empty slot zeroes the **gauges** too (`replicationLagBytes`, `memoryAllocBytes`) and has empty
  `sinkDelivery` / `sinkLatencyP95` records.
- `METRICS_POINTS_MAX` mirrors `pointsMax` in `apps/hump/internal/adapter/metrics_collector`.
  Change them together.

Two things are called sinks. `Metrics.sinks` is the sorted list of sink **types** that appear as
keys of `sinkDelivery` or `sinkLatencyP95` in any point. `topology.sinks` are sink **endpoints**,
one per type and host, with their targets.

## When it refetches

- **On a timer**: `refetchInterval` is a function, recomputed every time: the next `hh:mm:01`,
  strictly in the future (exactly at `:01` it answers 60 seconds). Same for every duration.
- **On live events**: `shared/state/invalidate.ts` invalidates the `['metrics']` prefix (every
  duration at once) for each live batch, and with `refetchType: 'all'` after a reconnect. That
  prefix match is why the query key must stay `keys.queries.metrics(duration)`.

`staleTime` is 10 seconds; `throwOnError: true` makes a failed request the query's error.

## Tests

`now` is a parameter of both `normalizeMetrics` and `refetchInterval` so the tests need no clock
mocking. The bigint cases cast `60n as unknown as number` to reproduce what the transformer hands
over. The hook itself is not tested.
