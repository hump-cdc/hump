# Chart

Internal building block of the charts, on top of `@tanstack/charts`: the sized frame, the
per-series colour variables, the legend, the tooltip card, the motion presets and the data
helpers. Consumed only by `charts/area-chart` and `charts/bar-chart`; application code imports
those (or the `card/*` wrappers), not this folder.

Exports (`index.ts`):

| Export | What it is |
|---|---|
| `ChartContainer` | frame: provides the config context, injects `ChartStyle`, renders the plot and the optional legend |
| `ChartStyle` | `<style>` that sets `--color-<key>` per series, scoped to `[data-chart=<id>]` |
| `ChartLegend` | row of swatches and labels read from the config |
| `ChartTooltipContent` | tooltip card: the x label, then one row per series |
| `chartTooltipOptions` | tooltip behaviour passed to `defineChart` |
| `useChartRenderer` | motion renderer: marks grow in on mount, spring between data updates, respect reduced motion |
| `chartSpring`, `chartCursorSpring` | spring of the marks; snappier spring of the hover cursor |
| `createChartConfig` | `ChartSeries[]` → `ChartConfig` (label, colour) |
| `toChartRows` | wide `ChartDataRow[]` → long `ChartRow[]`, with `start`/`end` stacked extents |
| `getChartMaximum` | top of the y domain, rounded up to a nice tick; `1` for an all-zero chart |
| `getSeriesColor` | `dataKey` → `var(--color-<dataKey>)` |
| `ChartConfig`, `ChartSeries`, `ChartDataRow`, `ChartRow`, `ChartVariant`, `ChartTooltipValue` | types |

## Usage

How `area-chart` and `bar-chart` compose it:

```tsx
import { defineChart } from '@tanstack/charts'
import { RendererChart } from '@tanstack/charts/react/tooltip'

import {
    ChartContainer,
    ChartTooltipContent,
    chartTooltipOptions,
    createChartConfig,
    getChartMaximum,
    getSeriesColor,
    toChartRows,
    useChartRenderer,
} from '@/shared/ui/charts/chart'

const config = createChartConfig(series)
const renderer = useChartRenderer()
const rows = toChartRows({ data, series, xAxisKey, stacked })
const definition = defineChart({
    marks: [/* marks over `rows`, painted with getSeriesColor(dataKey) */],
    scales: {/* y domain: [0, getChartMaximum(rows)] */},
    tooltip: chartTooltipOptions,
})

<ChartContainer config={config} legend={legend} className="aspect-auto h-40 w-full">
    {(chartId) => (
        <RendererChart
            definition={definition}
            renderer={renderer}
            idPrefix={chartId}
            renderTooltipBody={({ points }) => (
                <ChartTooltipContent points={points} seriesOrder={seriesOrder} valueProps={tooltipValue} />
            )}
        />
    )}
</ChartContainer>
```

- `ChartContainer` takes a render function, not JSX: it hands over the `chartId` the chart
  must pass as `idPrefix` so gradient ids stay unique per chart. An `id` prop replaces the
  generated id in `chart-<id>`.
- `ChartLegend` and `ChartTooltipContent` read the config from context and throw outside a
  `ChartContainer`. The context hook itself (`useChart`) is not exported.
- The container is `aspect-video` by default; the height comes from the consumer's `className`.

### Data (`toChartRows`)

These rules hold for every chart built on it:

- `data` is wide: one row per x position, one column per series `dataKey`. Rows are plotted by
  index; `xAxisKey` names the column shown as the tooltip heading. When its values are unique,
  they also key the marks, so a mark follows its label across updates; otherwise marks are
  keyed by index.
- A missing or non-numeric series value counts as `0`.
- `series` order is the stacking order; `label` falls back to `dataKey`.

## Props

- `ChartTooltipContent`: `seriesOrder` fixes the row order regardless of stacking; `valueProps`
  (`kind`, `suffix` of `NumberFlow`) applies to every value, and `kind` falls back to `integer`
  or `decimal` by value. Renders nothing without points.
- `ChartLegend`: `hideIcon` shows the colour swatch even when the config entry has an `icon`.
  Renders nothing for an empty config.
- `ChartStyle`: config entries without `color` or `theme` are skipped.

`createChartConfig` sets only `label` and `color`; nothing in the tree sets `icon` or `theme` of
a `ChartConfig` entry.

## Semantic styles

No cva variants. A series `variant` (`success`, `primary`, `danger`, `amber`) maps to the
token of the same name; `createChartConfig` resolves it and `ChartStyle` exposes it per chart as
`--color-<dataKey>`, which marks, legend and tooltip read.

- The container strips the library tooltip chrome through `--ts-chart-tooltip-*` (transparent,
  no border, shadow or padding), so `ChartTooltipContent` is the only visible surface.
