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

### Card

Renders a `div`; `HTMLAttributes<HTMLDivElement>` pass through.

| Prop | Type | Default | Description |
|---|---|---|---|
| `separated` | `boolean` | `true` | `true` — no card padding, header and footer are bordered bands; `false` — one even inset, no dividers |
| `asChild` | `boolean` | `false` | Render the child element through Radix `Slot` |
| `ref` | `Ref<HTMLDivElement>` | — | |

### CardHeader, CardFooter

Render a `div`; `HTMLAttributes<HTMLDivElement>` pass through. Both read `separated` from the
enclosing `Card`; outside a `Card` they render in the non-separated style.

| Prop | Type | Default | Description |
|---|---|---|---|
| `aside` | `ReactNode` | — | Right-aligned slot, wrapped in `flex items-center gap-2` |
| `ref` | `Ref<HTMLDivElement>` | — | |

### CardContent

A `div` with `flex-1`; `HTMLAttributes<HTMLDivElement>` pass through, plus `ref`. No own props.

### AreaChartCard, BarChartCard

No HTML attributes pass through. `data`, `series`, `xAxisKey`, `stacked`, `tooltip` and
`tooltipValue` go straight to `AreaChart` / `BarChart` (`@/shared/ui/charts/...`).

| Prop | Type | Default | Description |
|---|---|---|---|
| `title` | `string` | required | Header title |
| `subtitle` | `string` | — | Monospace note after the title |
| `hint` | `string` | — | Info icon with a tooltip next to the title |
| `legend` | `boolean` | `true` | Series legend in the header aside |
| `value` | `ReactNode` | — | **Area only.** Headline value above the chart; skipped for `null`, `undefined`, `''` |
| `data` | `Record<string, number \| string>[]` | required | One row per x position, one column per series |
| `series` | `{ dataKey: string; label?: string; variant: 'primary' \| 'success' \| 'danger' \| 'amber' }[]` | required | |
| `xAxisKey` | `string` | — | Column of `data` holding the x label |
| `stacked` | `boolean` | Area `false`, Bar `true` | |
| `tooltip` | `boolean` | `true` | Chart hover tooltip |
| `tooltipValue` | `{ kind?; suffix? }` (`NumberFlow` props) | — | Formatting of the values in the tooltip |
| `chartClassName` | `string` | `'h-56'` | Class of the chart, i.e. its height |
| `className` | `string` | — | On the card |
| `ref` | `Ref<HTMLDivElement>` | — | |

Legend: Area lists every series inline. Bar lists up to 2 inline; with more it collapses to up to
5 swatches plus `N series`, with the full list in a tooltip.

### MetricCard

No HTML attributes pass through.

| Prop | Type | Default | Description |
|---|---|---|---|
| `title` | `string` | required | Label above the value; also the series label of the sparkline |
| `hint` | `string` | — | Info icon with a tooltip next to the title |
| `value` | `ReactNode` | — | Headline value; skipped for `null`, `undefined`, `''` |
| `values` | `{ date: string; value: number }[]` | — | Sparkline data (a `BarChart`); no chart when omitted |
| `variant` | `'primary' \| 'success' \| 'danger' \| 'amber'` | `'primary'` | Colour of the sparkline bars; see Semantic styles |
| `aside` | `ReactNode` | — | Right of the title. `undefined` — automatic trend arrow; `null` — nothing |
| `asideTrend` | `'positive' \| 'negative' \| null` | `'positive'` | Which direction counts as good for the automatic arrow; `null` disables it |
| `tooltipValue` | `{ kind?; suffix? }` (`NumberFlow` props) | — | Formatting of the sparkline tooltip values |
| `chartSize` | `'sm' \| 'md' \| 'lg'` | `'sm'` | Sparkline height: `h-8` / `h-20` / `h-32` |
| `chartClassName` | `string` | — | Extra classes on the sparkline |
| `separated` | `boolean` | `false` | `true` — bordered `CardHeader` over a `p-4` body; `false` — one `p-5` block |
| `footer` | `ReactNode` | — | Small line under the chart |
| `className` | `string` | — | On the card |
| `ref` | `Ref<HTMLDivElement>` | — | |

Automatic trend arrow: compares the last two entries of `values`. Nothing is shown with fewer
than two entries or when they are equal. Up arrow when the last is higher, down when lower;
green when that direction matches `asideTrend`, red otherwise.

### InfoCard

Renders a `div`; `HTMLAttributes<HTMLDivElement>` pass through, except the native `title`.

| Prop | Type | Default | Description |
|---|---|---|---|
| `title` | `string` | required | Small uppercase heading |
| `variant` | `'neutral' \| 'success' \| 'failed' \| 'active' \| 'lag'` | `'neutral'` | Colour of the content |
| `footer` | `ReactNode` | — | Secondary line under the content |
| `children` | `ReactNode` | — | The content, monospace |
| `ref` | `Ref<HTMLDivElement>` | — | |

## Semantic styles

**Card.** `bg-surface-1`, `border-hairline`, `rounded-xl`, `text-ink`, `overflow-hidden`.

| `separated` | Card | Header | Footer |
|---|---|---|---|
| `true` | `gap-0 p-0` | `p-4`, `border-hairline` bottom | `p-4`, `border-hairline` top |
| `false` | `gap-3.5 p-4` | no padding, no border | no padding, no border |

The three chart cards add `overflow-visible` to the card.

**Series colour** (`variant` of a series, legend swatches): `primary` → `bg-primary`, `success` →
`bg-success`, `danger` → `bg-danger`, `amber` → `bg-amber`. Area swatches are short lines
(`h-1 w-5`), Bar swatches are squares (`size-2.5`).

**Chart card text.** Title `text-ink text-sm font-semibold`; subtitle and hint icon
`text-ink-tertiary`; legend `text-ink-subtle text-xs`; headline value
`font-mono font-semibold text-3xl text-ink`.

**MetricCard.** Title `text-ink-subtle text-sm`, footer `text-ink-subtle text-xs`. The `variant`
colours the sparkline; the headline value stays `text-ink` for `primary`, `success` and `amber`
and turns `text-danger` only for `danger`. Trend arrow: `text-success` good, `text-danger` bad.

**InfoCard.** `bg-surface-2/70`, `border-hairline`, `rounded-lg`, `p-3.5`; title
`text-ink-tertiary text-2xs uppercase tracking-widest`; footer `text-ink-subtle text-xs`. Content
is `font-mono font-medium text-md`, at least one line tall, coloured by `variant`:

| `variant` | Token | Meaning |
|---|---|---|
| `neutral` | `text-ink` | plain fact |
| `success` | `text-success` | ok, delivered |
| `failed` | `text-danger` | error, failures |
| `active` | `text-primary` | in progress, selected |
| `lag` | `text-amber` | warning, delayed |

None of the cards has hover, disabled or active states.
