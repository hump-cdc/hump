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

- The shape of `data` and `series` follows `charts/chart` ("Data").
- The size is `h-40 w-full` unless `className` overrides it (`MetricCard` passes `h-8`,
  `h-20` or `h-32`).
- Memoize `data` and `series`: the chart definition is rebuilt when either changes identity.

## Props

| Prop | Description |
|---|---|
| `stacked` | Default `true`, one bar per step; `false` puts the series side by side |
| `series` | May be empty: the tooltip then stays off even with `tooltip` |
| `tooltip`, `legend` | Both off by default; `BarChartCard` and `MetricCard` turn the tooltip on, and `BarChartCard` draws its own legend in the header |

## Semantic styles

No cva variants; series colours, tooltip and legend come from `charts/chart`.

- Bars are rounded on the outer end only.
- A step that is empty or shorter than a pixel gets a one-pixel stand-in bar, so it stays
  visible: in the first series' colour, or `--hairline-strong` when there is no series.
- Hover and keyboard focus select a whole x step: a `--surface-2` band behind it.
