# pages/statistics — how this page hangs together

`/statistics`. The page is only the arrangement: nine cards, each its own slice under
`entities/statistics/<name>` (`entities/statistics/AGENTS.md`), rendered **without props** in
`ui/statistics.tsx`. The page holds no state and fetches nothing.

## The time range is not on this page

The metric cards take their chart step from `useLiveControlsChartStep` (`entities/config/state`),
whose control is `LiveControls` in the app header. Do not add a range prop or a page-level store
for it; a new metric card reads the step the same way and only needs its place in the grid.

## Gotchas

- The column is capped at `max-w-6xl`, the same frame as `pages/snapshot`.
- The chart pair (`sink-delivery`, `sink-latency`) splits into two columns at `lg`, the small card
  pairs at `sm`: charts need the width.
