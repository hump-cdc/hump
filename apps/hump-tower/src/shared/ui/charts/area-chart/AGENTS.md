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

- The shape of `data` and `series` follows `charts/chart` ("Data").
- The size is `h-40 w-full` unless `className` overrides it.
- Memoize `data` and `series`: the chart definition is rebuilt when either changes identity.

## Props

| Prop | Description |
|---|---|
| `stacked` | Default `true`; `false` starts every series at zero, so the areas overlap |
| `tooltip`, `legend` | Both off by default; `AreaChartCard` turns the tooltip on and draws its own legend in the header |

## Semantic styles

No cva variants; series colours, tooltip and legend come from `charts/chart`.

- Each series is a 2px stroke over a vertical gradient of its colour, 35% opacity down to
  transparent.
- Hover and keyboard focus select a whole x position: a vertical `--hairline` rule and a dot on
  every series.
