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

Plus every prop of Radix `Checkbox.Root` (`checked`, `defaultChecked`, `onCheckedChange`,
`disabled`, `required`, `name`, `value`, `id`, …), passed through. Controlled or uncontrolled as
Radix defines it.

| Prop | Type | Default | Description |
|---|---|---|---|
| `invalid` | `boolean` | `false` | Error styling; also sets `aria-invalid` on the root |
| `className` | `string` | — | Merged through `cn` after the variant classes |
| `ref` | `Ref<HTMLButtonElement>` | — | Forwarded to the root `<button>` |

`checked="indeterminate"` shows a `Minus` instead of the `Check`.

## Semantic styles

Base: `size-4.5`, `rounded-[5px]`, 1px border, `bg-transparent`.

| `invalid` | Meaning | Tokens |
|---|---|---|
| `false` | normal | `border-hairline-strong` |
| `true` | the value fails validation | `border-danger`, focus ring `ring-danger` |

| State (`data-state`) | Tokens |
|---|---|
| `unchecked` | border from the `invalid` variant, no fill, no icon |
| `checked` | `bg-primary`, `border-primary`, icon in `text-on-primary` |
| `indeterminate` | `bg-surface-3`, `border-hairline-strong`, icon in `text-ink-muted` |

Other states:

- focus-visible: `ring-2 ring-primary` with `ring-offset-2 ring-offset-canvas`.
- disabled: `opacity-50`, `pointer-events-none`.

The root carries the `peer` class, so a sibling can style itself from the checkbox state.
