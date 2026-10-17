# BarChart

Axis-less bar chart: stacked or grouped bars per x step, for counts over discrete steps. Pick
`AreaChart` (`charts/area-chart`) for a continuous trend; pick `BarChartCard` or `MetricCard`
(`card`) when the chart needs a card with a title or a metric around it.
Storybook: `Charts/Bar Chart`.

## Usage

```tsx
import { BarChart } from '@/shared/ui/charts/bar-chart'

<BarChart
    className="h-20"
    data={[
        { date: '10:00', value: 46 },
        { date: '10:05', value: 62 },
    ]}
    series={[{ dataKey: 'value', label: 'Delivered', variant: 'primary' }]}
    xAxisKey="date"
    tooltip
    tooltipValue={{ kind: 'decimal', suffix: 'ms' }}
/>
```

- `data` is wide: one row per x step, one column per series `dataKey`. Rows are plotted by
  index; `xAxisKey` names the column shown as the tooltip heading. When its values are unique,
  they also key the marks, so a mark follows its label across updates; otherwise marks are keyed
  by index.
- The size is `h-40 w-full` unless `className` overrides it (`MetricCard` passes `h-8`,
  `h-20` or `h-32`).
- Memoize `data` and `series`: the chart definition is rebuilt when either changes identity.

## Props

Plus the `div` attributes and `ref`, passed to the container.

| Prop | Type | Default | Description |
|---|---|---|---|
| `data` | `ChartDataRow[]` (`Record<string, number \| string>[]`) | — | the rows; a missing or non-numeric series value counts as `0` |
| `series` | `ChartSeries[]` (`{ dataKey: string; label?: string; variant }`) | — | the plotted columns, in stacking order; `label` falls back to `dataKey`. May be empty |
| `xAxisKey` | `string` | — | column with the x label, shown in the tooltip |
| `stacked` | `boolean` | `true` | stacks the series in one bar; `false` puts them side by side |
| `tooltip` | `boolean` | `false` | shows the tooltip for the focused x step; stays off while `series` is empty |
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

- Bars are rounded on the outer end only.
- A step that is empty or shorter than a pixel gets a one-pixel stand-in bar, so it stays
  visible: in the first series' colour, or `--hairline-strong` when there is no series.
- Hover and keyboard focus select a whole x step: a band in `--surface-2` sits behind it.
- Tooltip and legend styling comes from `charts/chart`.
