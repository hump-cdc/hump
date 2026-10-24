# TextInput

A single-line text field with an optional leading icon, a built-in clear button and a trailing
slot. Use it for free text and search; for a SQL condition use `WhereInput`
(`@/shared/ui/where-input`), for a choice from a list `MultiComboBox`
(`@/shared/ui/multi-combo-box`).
Storybook: `Inputs/Text Input`.

## Usage

```tsx
import { Search } from 'lucide-react'

import { TextInput } from '@/shared/ui/text-input'

<TextInput
    value={search}
    onValueChange={setSearch}
    placeholder="Search payload, target, error…"
    icon={<Search />}
    font="sans"
/>
```

## Props

The props type in `text-input.tsx` is closed: other HTML attributes (`id`, `onBlur`, `aria-*`, …)
are not accepted. Not obvious from the types:

- `className` and `invalid` go to the wrapper `<div>`, so `aria-invalid` sits on the wrapper, not
  on the `<input>`. `ref` goes to the `<input>`.
- `value` (anything but `undefined`) makes it controlled; otherwise it starts from `defaultValue`.
- `onValueChange` fires on typing and on clear (`''`).
- `font` (`'mono'` default, or `'sans'`) sets the font of the typed text.
- `icon` gets its svg forced to `size-4`; `suffix` comes after the clear button and is neither
  sized nor coloured.

The clear button (`aria-label="Clear"`, `tabIndex={-1}`) shows whenever the value is non-empty and
the input is not disabled; it sets the value to `''` and refocuses the input.

## Semantic styles

No cva variants. The wrapper is `h-9 w-full`, so the width comes from the parent or `className`;
it is `border-hairline`, `bg-canvas`, with `text-ink` text and a `text-ink-tertiary` placeholder.

States, all on the wrapper:

- focus-within: `ring-2 ring-primary` with `ring-offset-2 ring-offset-canvas`.
- invalid: `border-danger`, focus ring `ring-danger`.
- disabled: `opacity-50`, `pointer-events-none`.
