# Chart

Internal building block of the charts, on top of `@tanstack/charts`: the sized frame, the
per-series colour variables, the legend, the tooltip card, the motion presets and the data
helpers. It has no story and no component of its own to render. Consumed only by
`charts/area-chart` and `charts/bar-chart`; application code imports those (or the `card/*`
wrappers), not this folder.

Exports (`index.ts`):

| Export | Kind | What it is |
|---|---|---|
| `ChartContainer` | component | frame: provides the config context, injects `ChartStyle`, renders the plot and the optional legend |
| `ChartStyle` | component | `<style>` that sets `--color-<key>` per series, scoped to `[data-chart=<id>]` |
| `ChartLegend` | component | row of swatches and labels read from the config |
| `ChartTooltipContent` | component | tooltip card: the x label, then one row per series |
| `chartTooltipOptions` | const | tooltip behaviour passed to `defineChart` |
| `useChartRenderer` | hook | motion renderer: marks grow in on mount, spring between data updates, respect reduced motion |
| `chartSpring`, `chartCursorSpring` | const | spring of the marks; snappier spring of the hover cursor |
| `createChartConfig` | function | `ChartSeries[]` → `ChartConfig` (label, colour) |
| `toChartRows` | function | wide `ChartDataRow[]` → long `ChartRow[]`, with `start`/`end` stacked extents |
| `getChartMaximum` | function | top of the y domain, rounded up to a nice tick; `1` for an all-zero chart |
| `getSeriesColor` | function | `dataKey` → `var(--color-<dataKey>)` |
| `ChartConfig`, `ChartSeries`, `ChartDataRow`, `ChartRow`, `ChartVariant`, `ChartTooltipValue` | types | see Props |

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
  must pass as `idPrefix` so gradient ids stay unique per chart.
- `ChartLegend` and `ChartTooltipContent` read the config from context and throw outside a
  `ChartContainer`. The context hook itself (`useChart`) is not exported.
- The container is `aspect-video` by default; the height comes from the consumer's `className`.

## Props

`ChartContainer` — plus the `div` attributes, except `children`. `id`, when set, replaces the
generated id in `chart-<id>`.

| Prop | Type | Default | Description |
|---|---|---|---|
| `config` | `ChartConfig` | — | series labels and colours, keyed by `dataKey` |
| `legend` | `boolean` | `false` | appends `ChartLegend` under the plot |
| `children` | `(chartId: string) => ReactNode` | — | renders the plot |

`ChartLegend` — plus the `div` attributes. Renders nothing for an empty config.

| Prop | Type | Default | Description |
|---|---|---|---|
| `hideIcon` | `boolean` | `false` | shows the colour swatch even when the config entry has an `icon` |

`ChartTooltipContent` — renders nothing without points.

| Prop | Type | Default | Description |
|---|---|---|---|
| `points` | `readonly ChartPoint<ChartRow, number, number>[]` | — | the focused points, from `renderTooltipBody` |
| `seriesOrder` | `string[]` | — | series keys; rows follow this order regardless of stacking |
| `valueProps` | `ChartTooltipValue` | — | `kind` and `suffix` for every value; `kind` falls back to `integer` or `decimal` by value |

`ChartStyle`

| Prop | Type | Default | Description |
|---|---|---|---|
| `id` | `string` | — | the chart id the rule is scoped to |
| `config` | `ChartConfig` | — | entries without `color` or `theme` are skipped |

Types:

| Type | Shape |
|---|---|
| `ChartVariant` | `'success' \| 'primary' \| 'danger' \| 'amber'` |
| `ChartSeries` | `{ dataKey: string; label?: string; variant: ChartVariant }` |
| `ChartDataRow` | `Record<string, number \| string>` — one row per x position, one column per series |
| `ChartRow` | `{ key; index; label; series; value; start; end }` — one row per x position and series |
| `ChartConfig` | `Record<string, { label?: ReactNode; icon?: ComponentType } & ({ color?: string } \| { theme: { light: string; dark: string } })>` |
| `ChartTooltipValue` | `Pick<NumberFlow props, 'kind' \| 'suffix'>` |

`createChartConfig` sets only `label` (falling back to `dataKey`) and `color`; nothing in the
tree sets `icon` or `theme`.

## Semantic styles

No cva variants. A series colour is its `ChartVariant`, mapped by `createChartConfig` and
exposed per chart as `--color-<dataKey>`:

| `variant` | Token |
|---|---|
| `success` | `--success` |
| `primary` | `--primary` |
| `danger` | `--danger` |
| `amber` | `--amber` |

- Container: `text-xs`; the library tooltip chrome is stripped through `--ts-chart-tooltip-*`
  (transparent background, no border, shadow or padding; text `--ink`), so the card below is
  the only visible surface.
- Tooltip card: `bg-surface-1`, `border-hairline`, `text-ink`, `shadow-xl`; series label
  `text-ink-subtle`; value `font-mono text-ink`, animated with `NumberFlow`.
- Legend: swatch `var(--color-<key>)`; a config `icon`, when present, is `text-ink-subtle`.
