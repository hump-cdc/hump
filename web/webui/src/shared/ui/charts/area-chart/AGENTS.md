# AreaChart

Axis-less area chart: one gradient-filled area with a stroke per series, for a trend over
ordered points. Pick `BarChart` (`charts/bar-chart`) for discrete steps; pick `AreaChartCard`
(`card`) when the chart needs a card with a title around it.
Storybook: `Charts/Area Chart`.

## Usage

```tsx
import { AreaChart } from '@/shared/ui/charts/area-chart'

<AreaChart
    className="h-32"
    data={[
        { month: 'Jan', delivered: 80, failed: 8 },
        { month: 'Feb', delivered: 96, failed: 12 },
    ]}
    series={[
        { dataKey: 'delivered', label: 'Delivered', variant: 'success' },
        { dataKey: 'failed', label: 'Failed', variant: 'danger' },
    ]}
    xAxisKey="month"
    tooltip
    legend
/>
```

- `data` is wide: one row per x position, one column per series `dataKey`. Rows are plotted
  by index; `xAxisKey` only names the column shown as the tooltip heading.
- The size is `h-40 w-full` unless `className` overrides it.
- Memoize `data` and `series`: the chart definition is rebuilt when either changes identity.

## Props

Plus the `div` attributes and `ref`, passed to the container.

| Prop | Type | Default | Description |
|---|---|---|---|
| `data` | `ChartDataRow[]` (`Record<string, number \| string>[]`) | — | the rows; a missing or non-numeric series value counts as `0` |
| `series` | `ChartSeries[]` (`{ dataKey: string; label?: string; variant }`) | — | the plotted columns, in stacking order; `label` falls back to `dataKey` |
| `xAxisKey` | `string` | — | column with the x label, shown in the tooltip |
| `stacked` | `boolean` | `true` | stacks the series; `false` starts every series at zero, so the areas overlap |
| `tooltip` | `boolean` | `false` | shows the tooltip for the focused x position |
| `tooltipValue` | `ChartTooltipValue` (`{ kind?; suffix? }` of `NumberFlow`) | — | how tooltip values are rendered; `kind` falls back to `integer` or `decimal` by value |
| `legend` | `boolean` | `false` | shows the legend under the plot |

## Semantic styles

No cva variants. The colour of a series is its `variant`:

| `variant` | Token |
|---|---|
| `success` | `--success` |
| `primary` | `--primary` |
| `danger` | `--danger` |
| `amber` | `--amber` |

- Each series: a 2px stroke in its colour over a vertical gradient of the same colour, from
  35% opacity down to transparent.
- Hover and keyboard focus select a whole x position: a vertical rule in `--hairline` follows
  it, and a dot in the series colour appears on every series.
- Tooltip and legend styling comes from `charts/chart`.
