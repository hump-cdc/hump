# ProgressBar

Plain horizontal bar showing completion from 0 to 100. Pick `ProgressChart`
(`charts/progress-chart`) when the bar needs a title and its percentage.
Storybook: `Placeholders/Progress Bar`.

## Usage

```tsx
import { ProgressBar } from '@/shared/ui/progress-bar'

<ProgressBar value={percent} />

<ProgressBar value={percent} size="sm" variant={status === 'done' ? 'success' : 'primary'} />
```

The bar is `w-full`; the parent sets the width. It renders `role="progressbar"` with
`aria-valuenow` (rounded), `aria-valuemin={0}` and `aria-valuemax={100}`, and sets no
accessible label of its own.

## Props

Plus the `div` attributes (except `children`) and `ref`; `className` goes to the track.

| Prop | Type | Default | Description |
|---|---|---|---|
| `value` | `number` | — | completion, 0 to 100; values outside are clamped |
| `variant` | `'primary' \| 'success' \| 'neutral'` | `'primary'` | colour of the fill |
| `size` | `'sm' \| 'md'` | `'md'` | height of the track |

## Semantic styles

Two cva sets: `size` on the track, `variant` on the fill.

| `variant` | Token | Meaning |
|---|---|---|
| `primary` | `bg-primary` | in progress (the snapshot tables use it until a table is done) |
| `success` | `bg-success` | completed |
| `neutral` | `bg-ink-subtle` | no call site yet |

| `size` | Height |
|---|---|
| `sm` | `h-2` |
| `md` | `h-2.5` |

- Track `bg-surface-3`, fully rounded.
- The fill width transitions over 500ms; the transition is off under `prefers-reduced-motion`.
