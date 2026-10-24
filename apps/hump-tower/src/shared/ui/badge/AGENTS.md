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

| Prop | Description |
|---|---|
| `border` | `true` — pill (`rounded-full`) with a visible tinted border; `false` — `rounded-md`, border in the fill colour |
| `dotAnimated` | Adds `animate-ping` to the dot; no effect without `dot` |
| `icon` | Any `svg` inside is sized `size-3.5`, shown at 60% opacity |
| `title` | Replaces the native `title` attribute: a dimmed (60% opacity) prefix before the children |

With `asChild` only `children` are rendered: `dot`, `icon` and `title` are ignored.

## Semantic styles

- `neutral` — no status, a plain tag; `healthy` — ok, success; `lag` — warning, delayed;
  `failed` — an error; `active` — in progress, selected.
- Each variant is a soft fill with its text colour (`bg-success-soft` / `text-success`, …).
  `border` adds a tinted border (`border-success/30`, `neutral`: `border-hairline`).
- The dot takes the text colour, except in `neutral`, where it is `bg-ink-subtle`.
- No hover, disabled or active state of its own.
