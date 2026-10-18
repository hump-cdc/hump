# entities/statistics — how this module hangs together

FSD `entities` layer. Nine slices, one per card of the Statistics page. Each slice is a
self-contained card: it fetches its own data through a hook and takes **no props**, so the page
only decides where it sits. The single consumer is `pages/statistics/ui/statistics.tsx`.

There is **no `entities/statistics/index.ts`**. Each slice has its own `index.ts` exporting exactly
one component; import `@/entities/statistics/<slice>`. Hooks and item types stay private.

## Layout

Every slice has the same three files:

| File | Role |
|---|---|
| `<slice>/model/use-<slice>.ts` | the hook: reads a query, returns chart-ready `values`, plus `value` (`useDelivered`, `useFailed`, `useHeapInUse`, `useReplicationLag`) or `series` (`useSinkDelivery`, `useSinkLatency`); `useThroughput` returns only `values`; `useTables` and `useSinks` return a plain `Item[]` of rows |
| `<slice>/ui/<slice>.tsx` | the card: calls the hook, hands the result to a `shared/ui/card` component |
| `<slice>/index.ts` | `export { <Component> } from './ui/<slice>'` |

There are no tests and no stories in this module.

## Slices

| Slice | Reads | Renders |
|---|---|---|
| `delivered` | `points[].delivered` | `MetricCard`, `success`; headline = **sum over the window** |
| `failed` | `points[].failed` | `MetricCard`, `danger`; headline = sum over the window |
| `throughput` | `points[].throughput` | `AreaChartCard`, `events/sec`, no headline |
| `replication-lag` | `points[].replicationLagBytes` | `MetricCard`, bytes; headline = **last point** |
| `heap-in-use` | `points[].memoryAllocBytes` | `AreaChartCard` titled "Memory", MB; headline = last point |
| `sink-delivery` | `sinks`, `points[].sinkDelivery` | `BarChartCard`, one stacked series per sink type |
| `sink-latency` | `sinks`, `points[].sinkLatencyP95` | `BarChartCard`, `P95(ms)`, one stacked series per sink type |
| `tables` | `config.tables` | plain `Card` list: `schema.name` + sink count, link to `/` |
| `sinks` | `config.sinks` | plain `Card` list: `Sink` (type + host), link to `/operations` |

`tables` and `sinks` read the **configuration**, not the metrics: they list what is configured and
show no counts, although the metrics response carries a topology of its own.

## Where the samples come from

The seven metric slices all do the same two calls:

```ts
const chartStep = useLiveControlsChartStep()   // entities/config/state/@x/statistics
const { data } = useMetrics(chartStep)         // entities/metrics/@x/statistics
```

- `useMetrics` is the `POST /api/metrics` query of `entities/metrics`. Its `select` runs
  `normalizeMetrics`, so a slice always gets `METRICS_POINTS_MAX` (20) points, oldest first, with a
  zero point standing in for every bucket the collector skipped, and `sinks` — every sink type
  seen across the points, sorted. Slices therefore never pad, sort or convert `bigint` themselves.
- All seven hooks pass the same `chartStep`, hence the same query key: one request feeds the page.
  Refresh is owned by `entities/metrics` (repoll at `hh:mm:01`) and `shared/state` (the metrics
  prefix is in `LIVE_EVENT_QUERIES`); nothing here schedules anything.
- `chartStep` is the **bucket width** (`1m` / `5m` / `10m`), not a look-back range. The step list
  in `entities/config/state` is hand-written and checked with `satisfies` against the generated
  request enum that `MetricsDuration` comes from.
- Sibling entities are reachable only through their `@x/statistics.ts` files. Each of those
  exposes the minimum: `useMetrics` from `entities/metrics`,
  `useLiveControlsChartStep` from `entities/config/state`, `useConfig` from
  `entities/config/config`. Need more — widen the `@x` file, do not import the entity's
  barrel or its files.

What the fields mean is decided by the backend, not here. Read
`apps/hump/internal/application/hump_tower/get_metrics/AGENTS.md` (the wire contract, `dto.go`) and
`apps/hump/internal/adapter/metrics_collector/AGENTS.md` ("Two clocks in one response", "What it
reads, and what it cannot") before changing a card's wording — `throughput`, for one, is
successfully delivered events per second, not rows read. No Prometheus metric name appears in this
module — slices see only `MetricsPoint` fields.

## Things that are easy to get wrong

- **`undefined` vs `0`.** While the query has no data, `value` is `undefined` and the card shows no
  headline. An empty response is different: it normalizes to twenty zero points, so the headline
  is `0`. Keep `value === undefined ? undefined : <NumberFlow …/>` in the card.
- **Two headline rules.** Counters (`delivered`, `failed`) sum the charted window, so the number
  changes with the chart step and is not an all-time total. Gauges (`replication-lag`,
  `heap-in-use`) take `points.at(-1)`, which is the newest bucket the response reported because
  the timeline ends there.
- **Trend arrow.** `MetricCard` draws one from the last two bars unless `aside` is passed.
  `delivered` and `failed` pass `aside={null}` to suppress it; `replication-lag` keeps it with
  `asideTrend="negative"`, so growing lag is the red direction.
- **Units are converted in the hook, labelled in the card.** `heap-in-use` turns bytes into MB
  (2 fraction digits) and the card appends `MB`; `sink-latency` and `throughput` round to 2
  digits; `replication-lag` passes raw bytes and lets `kind: 'bytes'` format them. Change one
  side and the tooltip `suffix` / `kind` in the card must follow.
- **Sink colours follow the sorted order.** `sink-delivery` and `sink-latency` assign
  `VARIANTS[index % 4]` over `data.sinks`; the sort is what keeps a sink's colour and stacking
  position stable. `VARIANTS` and `Series` are duplicated in the two hooks — no slice here
  imports a neighbour. A sink missing from a bucket is written as `0`, never left out.
- **`MetricCard` wants `{ date, value }`**; the area and bar cards take an `xAxisKey` and
  `series[].dataKey`, so those hooks name their keys (`time`, `eventsPerSec`, `heapInUse`, the sink
  type). X labels come from `formatTimeCompact` (local `HH:MM`) in every metric slice.
- `tables` and `sinks` rely on `ConfigProvider` having resolved the config before the page mounts;
  that is why they have no loading state, only the "No … configured" empty one.

## Adding a statistic

1. If it needs a new series: add the field to the response in `get_metrics/dto.go`, run
   `moon run hump-tower:openapi` (it only writes `openapi.json`), then `bun run generate:openapi` in
   `apps/hump-tower` to regenerate the TS client (`bun run dev` / `build` do it via `predev` /
   `prebuild`), then extend `MetricsPoint` in `entities/metrics/model/types.ts` together with
   `toPoint` **and** `emptyPoint` in `use-metrics.ts` — a padded bucket must have the field too.
2. Create `<slice>/model/use-<slice>.ts` (chart step → `useMetrics` → map `points`),
   `<slice>/ui/<slice>.tsx` (a prop-less `FC` over `MetricCard`, `AreaChartCard` or
   `BarChartCard`) and `<slice>/index.ts` exporting the component only.
3. Place it in the grid in `pages/statistics/ui/statistics.tsx`.
