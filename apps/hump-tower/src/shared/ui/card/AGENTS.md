# Card

The surface container of the UI kit and four ready-made cards built for one job each.

| Export | Storybook | Pick it for |
|---|---|---|
| `Card`, `CardHeader`, `CardContent`, `CardFooter` | `Content/Card` | any custom content in a framed surface |
| `AreaChartCard` | `Content/Card/Area` | a titled area chart with a legend, optionally a headline value |
| `BarChartCard` | `Content/Card/Bar` | a titled bar chart with a legend |
| `MetricCard` | `Content/Card/Metric` | one headline number, optionally with a sparkline and a trend arrow |
| `InfoCard` | `Content/Card/Info` | one small labelled fact in a details panel |

`AreaChartCard` and `BarChartCard` compose `Card` + `CardHeader` + `CardContent`; `MetricCard`
does so only when `separated`, otherwise its header is a plain `div` inside `Card`.
`InfoCard` is a standalone `div` and does not use `Card`. `card-context.ts` is internal.

## Usage

```tsx
import { Card, CardContent, CardFooter, CardHeader } from '@/shared/ui/card'

<Card>
    <CardHeader aside={<Badge variant="healthy" dot>Healthy</Badge>}>
        <span className="font-semibold text-ink text-md">Card title</span>
    </CardHeader>
    <CardContent>…</CardContent>
    <CardFooter aside={<Button variant="secondary">Action</Button>}>Footer</CardFooter>
</Card>
```

`CardHeader` and `CardFooter` put `children` on the left and `aside` on the right. `CardContent`
adds no padding in either mode — in a separated card pad the body yourself (`className="p-4"`).

```tsx
import { AreaChartCard, BarChartCard, InfoCard, MetricCard } from '@/shared/ui/card'

<MetricCard
    title="Delivered"
    hint="Change events successfully written to their destinations."
    value={<NumberFlow value={value} kind="integer" />}
    values={values}
    tooltipValue={{ kind: 'integer' }}
    variant="success"
/>

<AreaChartCard
    title="Replication lag"
    data={data}
    xAxisKey="time"
    series={[
        { dataKey: 'commit', label: 'commit', variant: 'primary' },
        { dataKey: 'sender', label: 'sender', variant: 'success' },
    ]}
/>

<InfoCard title="Failures" variant="failed">
    <NumberFlow value={stats?.failure} kind="compact" />
</InfoCard>
```

`hint` (on the three chart cards) renders a `Tooltip` from `@/shared/ui/tooltip`, which needs a
`TooltipProvider` above it — `Providers` and the Storybook preview both supply one.

## Props

### Card, CardHeader, CardFooter

`Card` `separated` (default `true`): `true` — no card padding, header and footer are bordered
bands; `false` — one even inset, no dividers. `CardHeader` and `CardFooter` read it from the
enclosing `Card`; outside a `Card` they render in the non-separated style.

### AreaChartCard, BarChartCard

No HTML attributes pass through. `data`, `series`, `xAxisKey`, `stacked`, `tooltip` and
`tooltipValue` go straight to `AreaChart` / `BarChart` (`@/shared/ui/charts/...`), but the
defaults differ from the raw charts:

| Prop | Card default | Chart default |
|---|---|---|
| `stacked` | Area `false`, Bar `true` | `true` |
| `tooltip` | `true` | `false` |
| `chartClassName` (the chart's height) | `'h-56'` | `h-40` |

`legend` (default `true`) is the card's own legend in the header aside, not the chart's: Area
lists every series inline. Bar lists up to 2 inline; with more it collapses to up to 5 swatches
plus `N series`, with the full list in a tooltip. `value` (Area only) and `MetricCard` `value`
are skipped for `null`, `undefined` and `''`.

### MetricCard

No HTML attributes pass through.

| Prop | Description |
|---|---|
| `title` | Label above the value; also the series label of the sparkline |
| `values` | Sparkline data (a `BarChart`); no chart when omitted |
| `aside` | Right of the title. `undefined` — automatic trend arrow; `null` — nothing |
| `asideTrend` | Default `'positive'`: which direction counts as good for the automatic arrow; `null` disables it |
| `chartSize` | Sparkline height: `sm` `h-8` (default) / `md` `h-20` / `lg` `h-32` |
| `separated` | Default `false`. `true` — bordered `CardHeader` over a `p-4` body; `false` — one `p-5` block |

Automatic trend arrow: compares the last two entries of `values`. Nothing is shown with fewer
than two entries or when they are equal. Up arrow when the last is higher, down when lower;
green when that direction matches `asideTrend`, red otherwise.

### InfoCard

`title` replaces the native attribute: a small uppercase heading. `children` is the content,
monospace and coloured by `variant`.

## Semantic styles

- `Card` `separated` — `true` for a header/footer as bordered bands over an unpadded body;
  `false` for one even inset with no dividers.
- The three chart cards add `overflow-visible` so the chart tooltip is not clipped.
- Series colours follow `charts/chart`. Area legend swatches are short lines, Bar swatches are
  squares.
- `MetricCard` `variant` colours the sparkline only; the headline value stays `text-ink` except
  for `danger`, where it turns `text-danger`. Trend arrow: `text-success` good, `text-danger` bad.
- `InfoCard` `variant`: `neutral` — a plain fact; `success` — ok, delivered; `failed` — an error,
  failures; `active` — in progress, selected; `lag` — warning, delayed. Its content is at least
  one line tall, so a number and the dash placeholder take the same height.
- None of the cards has hover, disabled or active states.
