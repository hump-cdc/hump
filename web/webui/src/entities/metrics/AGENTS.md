# entities/metrics — how this slice hangs together

The FSD `entities` slice that owns the dashboard's one metrics query: `useMetrics(duration)` over
`POST /api/metrics`, and the normalized shape every chart, the flows canvas and the operations
filter read. It has no `ui/` — it is a query and its types, nothing renders here. The backend half
is `internal/application/webui/get_metrics/AGENTS.md`; what the numbers mean is in
`internal/adapter/metrics_collector/AGENTS.md`, "Two clocks in one response".

## Layout

| File | Role |
|---|---|
| `model/types.ts` | the normalized shape — plain numbers, epoch milliseconds, full field names |
| `model/use-metrics.ts` | the query, its refresh schedule and `normalizeMetrics` with its converters |
| `model/use-metrics.test.ts` | `refetchInterval` and `normalizeMetrics`; the hook itself is not tested |
| `@x/<consumer>.ts` | the public API: one file per consuming entity |

Imports from `shared` are `@/shared/api` (the generated `Metrics.get` client and the `*Model`
types) and `@/shared/state` (`keys.queries.metrics`). Nothing from `shared/live`.

## Public API and who uses it

There is **no `index.ts`**. Every consumer is another entity, and entities reach each other only
through `@x`, so the `@x` files are the whole public API. `normalizeMetrics`, `refetchInterval`
and `METRICS_POINTS_MAX` are exported from `use-metrics.ts` for the test only and are in no `@x`
file. The types in `model/types.ts` that no `@x` file re-exports (`Metrics`, `MetricsTableOp`,
`MetricsSinkTarget`, `MetricsDuration`) are internal.

| File | Consumer | Exposes |
|---|---|---|
| `@x/statistics.ts` | the seven chart slices of `entities/statistics` (`delivered`, `failed`, `throughput`, `heap-in-use`, `replication-lag`, `sink-delivery`, `sink-latency`) | `useMetrics`, `MetricsPoint` |
| `@x/flows.ts` | `entities/flows/flows` | `useMetrics`, `MetricsTopology`, `MetricsSink` |
| `@x/inspector.ts` | `entities/flows/inspector` | `useMetrics`, `MetricsTopology`, `MetricsTable`, `MetricsDelivery`, `MetricsConnection` |
| `@x/operations.ts` | `entities/operations/filter` | `useMetrics` |

The files are named after the consumer, and not consistently by group or by slice: `statistics`
and `operations` are group folders, `flows` and `inspector` are two slices of the `flows` group.
A new consuming entity gets **its own** `@x/<consumer>.ts` re-exporting from `../model/*` only
what it needs; do not point it at an existing file, and do not widen a file for a different
consumer. A consumer from a layer above `entities` would need an `index.ts` added first — the
`biome.json` overrides let those layers import a slice through its barrel only.

Every consumer calls `useMetrics` with the chart step from the live controls
(`useLiveControlsChartStep`, `entities/config/state`), so they share one cache entry and one
request.

## From response to `Metrics`

The cache holds the **raw response**; `normalizeMetrics` runs in `select`. Reading the cache
directly through `keys.queries.metrics(duration)` gives a `MetricsResponseBodyModel`, not a
`Metrics`.

- **Counters are `bigint` at runtime.** The generated client parses responses with zod, which
  coerces `int64` fields to `bigint`, while the generated types say `number`. `toNumber` /
  `toNumbers` exist for that: `bucketSeconds`, `delivered`, `failed`, `sinkDelivery`, every `n`,
  `operations`, and a connection's `success` / `failure` go through them. Floats (`throughput`,
  the gauges, the other `p95` fields, the `success` share) pass straight through, except
  `sinkLatencyP95`: a float map (`map[string]float64`, `z.number()`) that `toPoint` still runs
  through `toNumbers`, a no-op for it. A new integer field needs
  the conversion even though TypeScript will not ask for it.
- **The wire abbreviations stop here.** `i` / `u` / `d` / `t` become `insert` / `update` /
  `delete` / `truncate`, `n` becomes `count`, and the `tables` map becomes an array with the key
  as `name`.
- **Absent stays absent.** `p95` and `success` are optional and are copied as `undefined`, never
  defaulted to 0; a sink type without an estimate is simply missing from `sinkLatencyP95`.
- **Topology is sorted** so a layout does not move between refetches: tables by name, sinks by
  type then host, targets by name, with a locale-independent comparison. `tables` arrives as a
  map; `connections` keep the response order.

`MetricsDuration` is not declared here — it is the `duration` field of the generated request
type, so a bucket width added on the backend arrives with the regenerated client. `CHART_STEPS`
in `entities/config/state` is a separate hand-written list, checked against this type with
`satisfies`.

## The fixed timeline

`points` always has `METRICS_POINTS_MAX` (20) entries, oldest first, `bucketMs` apart. The
response is not trusted to be contiguous: points are looked up by exact timestamp, and a slot
with no match becomes `emptyPoint`, so a bucket the collector skipped is a zero bar instead of
shifting the bars after it.

- The timeline ends at the **newest point of the response**, not at the current time. Only an
  empty response falls back to `now`, floored to the bucket width.
- The lookup is exact, so a point that is not a whole number of buckets before the newest one is
  dropped.
- An empty slot zeroes the **gauges** too (`replicationLagBytes`, `memoryAllocBytes`), not just
  the counters, and has empty `sinkDelivery` / `sinkLatencyP95` records.
- `METRICS_POINTS_MAX` mirrors `pointsMax` in `internal/adapter/metrics_collector`. Change them
  together.

Two things are called sinks. `Metrics.sinks` is the sorted list of sink **types** that appear as
keys of `sinkDelivery` or `sinkLatencyP95` in any point. `topology.sinks` are sink **endpoints**,
one per type and host, with their targets.

## When it refetches

- **On a timer**: `refetchInterval` is a function, so the delay is recomputed every time —
  the next `hh:mm:01`, strictly in the future (exactly at `:01` it answers 60 seconds). The
  schedule is the same for every duration.
- **On live events**: `shared/state/invalidate.ts` invalidates the `['metrics']` prefix — every
  duration at once — for each live batch, and again with `refetchType: 'all'` after the stream
  reconnects. `LiveUpdates` (`shared/live`, mounted in `src/app/providers.tsx`) drives both. That
  prefix match is why the query key must stay `keys.queries.metrics(duration)`.

`queryFn` passes the query's `signal` to the client and uses `throwOnError: true`, so a failed
request surfaces as the query's error. `staleTime` is 10 seconds.

## Tests

`bun test`. `point` and `response` build wire models with zeros and empty collections
(`response` leaves `points: null`), and the bigint cases cast `60n as unknown as number` to
reproduce what the transformer hands over. `now` is a parameter of both `normalizeMetrics` and
`refetchInterval` precisely so the tests need no clock mocking.
