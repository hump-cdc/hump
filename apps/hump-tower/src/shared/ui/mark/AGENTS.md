# Mark

An inline monospace chip for a keyword, identifier or value inside running text (`WHERE`, a table
name). For a multi-line or highlighted snippet pick `@/shared/ui/code`; for a status label pick
`@/shared/ui/badge`.
Storybook: `Content/Mark`.

## Usage

```tsx
import { Mark } from '@/shared/ui/mark'

<p className="text-ink-muted text-sm">
    Add an optional <Mark>WHERE</Mark> condition per table to snapshot only matching rows.
</p>
```

## Props

No props of its own: it renders a `<mark data-slot="mark">` and passes the `HTMLElement` attributes
and `ref` through.

## Semantic styles

No cva variants and no states. One fixed look: `bg-surface-3` fill, `border-hairline` border,
`text-ink`, `font-mono`. Font size is inherited from the surrounding text.
