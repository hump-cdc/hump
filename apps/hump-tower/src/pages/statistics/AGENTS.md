# pages/statistics — how this page hangs together

`/statistics`, mounted by `app/routes/_sidebar/statistics.tsx`, which sets only `component` and
`staticData.title`. No search params, no loader.

## Layout

| File | Role |
|---|---|
| `index.ts` | public contract: `Statistics` |
| `ui/statistics.tsx` | the card grid: order, grouping and breakpoints |

## The page is the arrangement

Nine cards, each its own slice under `entities/statistics/<name>`, imported one by one (there is
no `entities/statistics` barrel) and rendered **without props**:

| Row | Cards | Columns |
|---|---|---|
| 1 | `delivered`, `failed` | 2 from `sm` |
| 2 | `throughput` | full width |
| 3 | `replication-lag` | full width |
| 4 | `sink-delivery`, `sink-latency` | 2 from `lg` |
| 5 | `heap-in-use` | full width |
| 6 | `tables`, `sinks` | 2 from `sm` |

The page holds no state and fetches nothing. Each card has its own `model/use-*.ts`: seven read
the shared metrics query (`useMetrics` from `entities/metrics`) and `tables` / `sinks` read the
config query (`useConfig` from `entities/config/config`). See `entities/statistics/AGENTS.md`.

## The time range is not on this page

The metric cards take their chart step from `useLiveControlsChartStep` (`entities/config/state`),
whose control is `LiveControls` in the app header (`widgets/sidebar/app-header.tsx`). There is
nothing to thread through here — do not add a range prop or a page-level store for it; a new
metric card reads the step the same way and then only needs its place in the grid.

## Gotchas

- The column is capped at `max-w-6xl` — the same frame as `pages/snapshot`.
- The chart pair in row 4 splits at `lg`, the small cards at `sm`: charts need the width.
