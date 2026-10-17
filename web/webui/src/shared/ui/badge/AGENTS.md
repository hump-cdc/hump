# Badge

A small inline label for a status, a tag or a `title: value` pair. Use it directly for free-form
labels; for an operation kind or an operation status pick `@/shared/ui/operation-badge` or
`@/shared/ui/operation-status-badge`, which wrap it with a fixed mapping.
Storybook: `Content/Badge`.

## Usage

```tsx
import { Database } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'

<Badge variant="healthy" dot>Healthy</Badge>
<Badge icon={<Database />} title="db">app</Badge>
<Badge variant="active" border dot dotAnimated>live</Badge>
```

Rendered order inside the badge: dot, icon, title, children.

## Props

Renders a `<span>`; other `HTMLAttributes<HTMLSpanElement>` pass through. The native `title`
attribute is replaced by the `title` prop below.

| Prop | Type | Default | Description |
|---|---|---|---|
| `variant` | `'neutral' \| 'healthy' \| 'lag' \| 'failed' \| 'active'` | `'neutral'` | Semantic colour, see below |
| `border` | `boolean` | `false` | `true` — pill (`rounded-full`) with a visible tinted border; `false` — `rounded-md`, border in the fill colour |
| `dot` | `boolean` | `false` | Leading status dot |
| `dotAnimated` | `boolean` | `false` | Adds `animate-ping` to the dot; no effect without `dot` |
| `icon` | `ReactNode` | — | Leading icon; any `svg` inside is sized `size-3.5`, shown at 60% opacity |
| `title` | `ReactNode` | — | Dimmed (60% opacity) prefix before the children |
| `asChild` | `boolean` | `false` | Render the child element through Radix `Slot` |
| `ref` | `Ref<HTMLSpanElement>` | — | |

With `asChild` only `children` are rendered: `dot`, `icon` and `title` are ignored.

## Semantic styles

| `variant` | Meaning | Fill / text | Border, `border={false}` | Border, `border` |
|---|---|---|---|---|
| `neutral` | no status, plain tag | `bg-surface-3` / `text-ink-muted` | `border-surface-3` | `border-hairline` |
| `healthy` | ok, success | `bg-success-soft` / `text-success` | `border-success-soft` | `border-success/30` |
| `lag` | warning, delayed | `bg-amber-soft` / `text-amber` | `border-amber-soft` | `border-amber/30` |
| `failed` | error | `bg-danger-soft` / `text-danger` | `border-danger-soft` | `border-danger/30` |
| `active` | in progress, selected | `bg-primary-soft` / `text-primary` | `border-primary-soft` | `border-primary/30` |

The dot takes the text colour (`bg-current`), except in `neutral`, where it is `bg-ink-subtle`.
The badge is `select-none`, `whitespace-nowrap`, `text-sm font-medium`, at least `min-h-6` tall,
and has no hover, disabled or active state of its own.
