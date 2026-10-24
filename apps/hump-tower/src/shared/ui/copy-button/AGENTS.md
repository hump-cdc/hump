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

Only `value`, `label` and `className`; no other `<button>` attribute or `ref` is accepted.
`label` is the `aria-label` and the button's only accessible name.

Copying goes through `useClipboard()` from `@/shared/utils` (`navigator.clipboard.writeText`).
`copied` stays `true` for 1500 ms after a successful write; a rejected write is swallowed and
leaves the icon unchanged.

## Semantic styles

No cva variants. Idle: `Copy` icon in `text-ink-tertiary`, hover `text-ink`. Copied: `Check`
icon in `text-success`. There is no focus-visible ring and no disabled state.
