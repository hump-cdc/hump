# OperationStatusBadge

The terminal delivery outcome of an operation — delivered or failed — as a labelled badge or a bare
icon. Use `variant="compact"` in dense table cells and the default `full` elsewhere; for the
operation kind (insert, update, …) use `OperationBadge` (`@/shared/ui/operation-badge`).
Storybook: `Content/Operation Status Badge`.

## Usage

```tsx
import { OperationStatusBadge } from '@/shared/ui/operation-status-badge'

<OperationStatusBadge status={operation.status} attempts={operation.attempts} />
<OperationStatusBadge status={operation.status} attempts={operation.attempts} withAttempts />
<OperationStatusBadge variant="compact" status={operation.status} attempts={operation.attempts} />
```

## Props

No other props are accepted: HTML attributes and `Badge` props are not passed through.

| Prop | Type | Default | Description |
|---|---|---|---|
| `status` | `'delivered' \| 'failed'` | required | the outcome |
| `attempts` | `number` | required | delivery attempts; more than 1 means it was retried |
| `variant` | `'full' \| 'compact'` | `'full'` | `full` is a labelled `Badge`, `compact` a bare icon |
| `withAttempts` | `boolean` | — | `full` only: spells the count out in the label (`after N attempts`) instead of `after retry` |
| `className` | `string` | — | goes to the `Badge` (`full`) or the icon (`compact`) |

## Semantic styles

`variant` is a plain prop, not a cva variant; the look follows from `status` and `attempts`.

`full` — a `Badge` (`@/shared/ui/badge`) with a `size-3` icon:

| `status` | `attempts` | `Badge` variant | Tokens | Icon | Label |
|---|---|---|---|---|---|
| `delivered` | 1 or less | `healthy` | `bg-success-soft text-success` | `CircleCheck` | `Delivered` |
| `delivered` | above 1 | `lag` | `bg-amber-soft text-amber` | `RotateCcw` | `Delivered after retry` / `Delivered after N attempts` |
| `failed` | 1 or less | `failed` | `bg-danger-soft text-danger` | `CircleX` | `Failed` |
| `failed` | above 1 | `failed` | `bg-danger-soft text-danger` | `CircleX` | `Failed after retry` / `Failed after N attempts` |

`compact` — an icon only (`size-3.75`), `attempts` and `withAttempts` are ignored:

| `status` | Icon | Token | `aria-label` |
|---|---|---|---|
| `delivered` | `CircleCheck` | `text-success` | `Delivered` |
| `failed` | `CircleX` | `text-danger` | `Failed` |
