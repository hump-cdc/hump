# CopyButton

An icon-only button that copies a string to the clipboard and confirms it by swapping the icon.
Pick it next to a value the user may want to copy; for any other action use `Button`
(`@/shared/ui/button`).
Storybook: `Actions/Copy Button`.

## Usage

```tsx
import { CopyButton } from '@/shared/ui/copy-button'

<CopyButton value={query} label="Copy query" />
```

It has no size or box of its own. A caller that needs a framed button adds it through `className`
(`entities/operations/details`):

```tsx
<CopyButton
    value={payload}
    label="Copy payload"
    className="absolute top-3 right-4 size-6.5 rounded-md border border-hairline bg-surface-2"
/>
```

## Props

Only these three; no other `<button>` attribute or `ref` is accepted.

| Prop | Type | Default | Description |
|---|---|---|---|
| `value` | `string` | required | The text written to the clipboard on click |
| `label` | `string` | required | `aria-label` of the button; the only accessible name it has |
| `className` | `string` | — | Merged through `cn` after the base classes |

Copying goes through `useClipboard()` from `@/shared/utils` (`navigator.clipboard.writeText`).
`copied` stays `true` for 1500 ms after a successful write; a rejected write is swallowed and
leaves the icon unchanged.

## Semantic styles

No cva variants.

| State | Icon | Tokens |
|---|---|---|
| idle | `Copy`, `size-3.25` | `text-ink-tertiary`, hover `text-ink` |
| copied | `Check`, `size-3.25` | `text-success` on the icon |

There is no focus-visible ring and no disabled state.
