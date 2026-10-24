# NumberFlow

An animated number (`@number-flow/react`) that formats a raw value by `kind` and animates the
digits when it changes. Use it wherever a live metric is shown; for time elapsed since a start
date pick `@/shared/ui/elapsed-time`, which ticks by itself.
Storybook: `Content/Number Flow`.

`index.ts` exports `NumberFlow` only. `duration-flow.tsx` (`DurationFlow`) is internal, reached as
`<NumberFlow kind="duration">`.

## Usage

```tsx
import { NumberFlow } from '@/shared/ui/number-flow'

<span className="font-semibold text-3xl text-ink tracking-tight">
    <NumberFlow value={stats?.operations} kind="compact" />
</span>
<NumberFlow value={stats?.p95} kind="latency" />
<NumberFlow value={seconds} kind="duration" />
```

## Props

`value` is `number | undefined`: `undefined` renders a dash (`—`) of the same height, for every
`kind`. `kind` defaults to `'integer'`; `suffix` is appended after the kind's own suffix.

| `kind` | Input | Output |
|---|---|---|
| `integer` | number | no fraction digits, thousands grouped with a non-breaking space |
| `decimal` | number | up to 2 fraction digits, no grouping |
| `compact` | number | compact notation (`1.28M`); exactly 2 fraction digits from `999.995` up, at most 2 below |
| `percent` | percent value, already 0–100 | no fraction digits + `%`; the value is not multiplied |
| `bytes` | bytes | scaled by 1024 to `B`/`KB`/`MB`/`GB`/`TB`; 0 fraction digits for `B`, 2 otherwise |
| `latency` | milliseconds | `ms` rounded below 1000, otherwise seconds with 2 fraction digits and `s` |
| `duration` | seconds | `Xh Ym Zs`, leading zero units dropped (`Ym Zs`, `Zs`); rounded, clamped at 0; the units animate as one `NumberFlowGroup` |

The scalar kinds mirror the string formatters in `@/shared/utils` (`formatBytes`, `formatLatency`),
so an animated value reads the same as a formatted one. `bytes` and `latency` scale through
`scaleBytes` / `scaleLatency`, which return `{ value, suffix, fractionDigits }` instead of a string.

## Semantic styles

No cva variants and no colour tokens. Both parts add only `font-mono`; size, weight and colour are
inherited, so set the typography on a parent or through `className`. The dash placeholder is
`inline-block leading-none` with the vertical padding NumberFlow puts around its digits, so a
value and its placeholder share one height.
