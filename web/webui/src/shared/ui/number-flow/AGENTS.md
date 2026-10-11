# NumberFlow

An animated number (`@number-flow/react`) that formats a raw value by `kind` and animates the
digits when it changes. Use it wherever a live metric is shown; for time elapsed since a start
date pick `@/shared/ui/elapsed-time`, which ticks by itself.
Storybook: `Content/Number Flow`.

Parts:

| File | Exports | Public |
|---|---|---|
| `number-flow.tsx` | `NumberFlow` | yes, through `index.ts` |
| `duration-flow.tsx` | `DurationFlow` | no — `index.ts` does not re-export it; reached as `<NumberFlow kind="duration">` |

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

### NumberFlow

| Prop | Type | Default | Description |
|---|---|---|---|
| `value` | `number \| undefined` | required | `undefined` renders a dash (`—`) of the same height, for every `kind` |
| `kind` | `'integer' \| 'decimal' \| 'compact' \| 'percent' \| 'bytes' \| 'latency' \| 'duration'` | `'integer'` | How the value is read and formatted, see below |
| `suffix` | `string` | — | Appended after the kind's own suffix |
| `className` | `string` | — | |

| `kind` | Input | Output |
|---|---|---|
| `integer` | number | no fraction digits, thousands grouped with a non-breaking space |
| `decimal` | number | up to 2 fraction digits, no grouping |
| `compact` | number | compact notation (`1.28M`); exactly 2 fraction digits from `999.995` up, at most 2 below |
| `percent` | percent value, already 0–100 | no fraction digits + `%`; the value is not multiplied |
| `bytes` | bytes | scaled by 1024 to `B`/`KB`/`MB`/`GB`/`TB`; 0 fraction digits for `B`, 2 otherwise |
| `latency` | milliseconds | `ms` rounded below 1000, otherwise seconds with 2 fraction digits and `s` |
| `duration` | seconds | delegated to `DurationFlow` |

The scalar kinds mirror the string formatters in `@/shared/utils` (`scaleBytes`, `scaleLatency`),
so an animated value reads the same as a formatted one.

### DurationFlow (internal)

Renders `Xh Ym Zs`, dropping leading zero units (`Ym Zs`, `Zs`); the units animate as one
`NumberFlowGroup`. Seconds are rounded and clamped at 0.

| Prop | Type | Default | Description |
|---|---|---|---|
| `seconds` | `number` | required | |
| `suffix` | `string` | — | Appended after the trailing `s` |
| `className` | `string` | — | |

## Semantic styles

No cva variants and no colour tokens. Both parts add only `font-mono`; size, weight and colour are
inherited, so set the typography on a parent or through `className`. The dash placeholder is
`inline-block leading-none` with the vertical padding NumberFlow puts around its digits, so a
value and its placeholder share one height.
