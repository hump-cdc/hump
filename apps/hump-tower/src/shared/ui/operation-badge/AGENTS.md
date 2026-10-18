# OperationBadge

A `Badge` (`@/shared/ui/badge`) for a row operation — insert, update, delete, truncate — with the
colour fixed per operation. Use it wherever an operation kind is shown; for a delivery outcome use
`OperationStatusBadge` (`@/shared/ui/operation-status-badge`).
Storybook: `Content/Operation Badge`.

## Usage

```tsx
import { OperationBadge } from '@/shared/ui/operation-badge'

<OperationBadge op="insert" />
```

## Props

Plus every `Badge` prop except `variant` and `children`.

| Prop | Type | Default | Description |
|---|---|---|---|
| `op` | `'insert' \| 'update' \| 'delete' \| 'truncate'` | required | the operation; it is both the badge text and the colour key |
| `className` | `string` | — | merged after the component's own `px-1 uppercase leading-4` |

## Semantic styles

No variants of its own: `op` maps to a `Badge` variant. The text is the `op` value, rendered
uppercase with tighter padding than a plain badge (`px-1 leading-4`).

| `op` | `Badge` variant | Tokens |
|---|---|---|
| `insert` | `healthy` | `bg-success-soft text-success` |
| `update` | `active` | `bg-primary-soft text-primary` |
| `delete` | `failed` | `bg-danger-soft text-danger` |
| `truncate` | `lag` | `bg-amber-soft text-amber` |
