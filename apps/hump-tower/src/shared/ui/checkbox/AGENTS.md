# Checkbox

A tri-state checkbox (unchecked, checked, indeterminate) over `@radix-ui/react-checkbox`. Use it
for a boolean or a "select all" control; the rows of `Table` (`@/shared/ui/table`) render it
themselves.
Storybook: `Inputs/Checkbox`.

## Usage

```tsx
import { Checkbox } from '@/shared/ui/checkbox'

<label htmlFor="snapshot-select-all" className="flex items-center gap-2">
    <Checkbox
        id="snapshot-select-all"
        checked={selectAllState} // boolean | 'indeterminate'
        onCheckedChange={toggleAll}
        disabled={rows.length === 0}
    />
    <span>Select all</span>
</label>
```

The component renders only the box. The caller supplies the label (`<label htmlFor>` or
`aria-label`).

## Props

Every prop of Radix `Checkbox.Root` passes through; controlled or uncontrolled as Radix defines it.

| Prop | Description |
|---|---|
| `invalid` | Error styling; also sets `aria-invalid` on the root |

`checked="indeterminate"` shows a `Minus` instead of the `Check`.

## Semantic styles

- `invalid` — the value fails validation: `border-danger` and a `ring-danger` focus ring.
- `indeterminate` is deliberately quieter than `checked`: `bg-surface-3` with a `text-ink-muted`
  icon, against `bg-primary` for checked.
- The root carries the `peer` class, so a sibling can style itself from the checkbox state.
