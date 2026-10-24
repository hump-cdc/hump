# entities/statistics — how this module hangs together

FSD `entities` layer. Nine slices, one per card of the Statistics page. Each slice is a
self-contained card: its hook fetches the data, and the component takes **no props**, so the page
only decides where it sits. Each slice's `index.ts` exports exactly one component; hooks and item
types stay private. There is no `entities/statistics/index.ts`.

## Slices

| Slice | Reads | Renders |
|---|---|---|
| `delivered` | `points[].delivered` | `MetricCard`; headline = **sum over the window** |
| `failed` | `points[].failed` | `MetricCard`; headline = sum over the window |
| `throughput` | `points[].throughput` | `AreaChartCard`, `events/sec`, no headline |
| `replication-lag` | `points[].replicationLagBytes` | `MetricCard`, bytes; headline = **last point** |
| `heap-in-use` | `points[].memoryAllocBytes` | `AreaChartCard` titled "Memory", MB; headline = last point |
| `sink-delivery` | `sinks`, `points[].sinkDelivery` | `BarChartCard`, one stacked series per sink type |
| `sink-latency` | `sinks`, `points[].sinkLatencyP95` | `BarChartCard`, `P95(ms)`, one stacked series per sink type |
| `tables` | `config.tables` | plain `Card` list: `schema.name` + sink count, link to `/` |
| `sinks` | `config.sinks` | plain `Card` list: type + host, link to `/operations` |

`tables` and `sinks` read the **configuration**, not the metrics: they list what is configured and
show no counts.

## Where the samples come from

The seven metric hooks read `useLiveControlsChartStep()` and pass it to `useMetrics(chartStep)`,
both through `@x/statistics.ts` files of `entities/config/state` and `entities/metrics`.

- `useMetrics` already normalizes: always `METRICS_POINTS_MAX` (20) points, oldest first, a zero
  point for every skipped bucket, `bigint` converted, and `sinks` sorted. Slices never pad, sort or
  convert themselves.
- All seven pass the same `chartStep`, so one request feeds the page. Refresh is owned by
  `entities/metrics`; nothing here schedules anything.
- `chartStep` is the **bucket width** (`1m` / `5m` / `10m`), not a look-back range.

What the fields mean is decided by the backend: read
`apps/hump/internal/application/hump_tower/get_metrics/AGENTS.md` and
`apps/hump/internal/adapter/metrics_collector/AGENTS.md` ("Two clocks in one response", "What it
reads, and what it cannot") before changing a card's wording. `throughput`, for one, is
successfully delivered events per second, not rows read.

## Things that are easy to get wrong

- **`undefined` vs `0`.** While the query has no data, `value` is `undefined` and the card shows no
  headline. An empty response normalizes to twenty zero points, so the headline is `0`. Keep
  `value === undefined ? undefined : <NumberFlow …/>` in the card.
- **Two headline rules.** Counters (`delivered`, `failed`) sum the charted window, so the number
  changes with the chart step and is not an all-time total. Gauges (`replication-lag`,
  `heap-in-use`) take `points.at(-1)`, the newest bucket the response reported.
- **Trend arrow.** `MetricCard` draws one from the last two bars unless `aside` is passed.
  `delivered` and `failed` pass `aside={null}`; `replication-lag` keeps it with
  `asideTrend="negative"`, so growing lag is the red direction.
- **Units are converted in the hook, labelled in the card.** `heap-in-use` turns bytes into MB
  (2 fraction digits) and the card appends `MB`; `sink-latency` and `throughput` round to 2
  digits; `replication-lag` passes raw bytes to `kind: 'bytes'`. Change one side and the tooltip
  `suffix` / `kind` in the card must follow.
- **Sink colours follow the sorted order.** `sink-delivery` and `sink-latency` assign
  `VARIANTS[index % 4]` over `data.sinks`; the sort keeps a sink's colour and stacking position
  stable. `VARIANTS` and `Series` are duplicated in the two hooks on purpose (no slice imports a
  neighbour). A sink missing from a bucket is written as `0`, never left out.
- **Chart data keys.** `MetricCard` wants `{ date, value }`; the area and bar cards take an
  `xAxisKey` and `series[].dataKey`, so those hooks name their keys (`time`, `eventsPerSec`,
  `heapInUse`, the sink type).
- `tables` and `sinks` rely on `ConfigProvider` having loaded the config before the page mounts,
  so they have no loading state, only the "No … configured" empty one.

## Adding a statistic

1. If it needs a new series: add the field in `get_metrics/dto.go`, run
   `moon run hump-tower:openapi` and `bun run generate:openapi`, then extend `MetricsPoint` in
   `entities/metrics/model/types.ts` together with `toPoint` **and** `emptyPoint` in
   `use-metrics.ts`: a padded bucket must have the field too.
2. Create `<slice>/model/use-<slice>.ts`, `<slice>/ui/<slice>.tsx` (a prop-less `FC` over
   `MetricCard`, `AreaChartCard` or `BarChartCard`) and `<slice>/index.ts` exporting the component
   only.
3. Place it in the grid in `pages/statistics/ui/statistics.tsx`.
