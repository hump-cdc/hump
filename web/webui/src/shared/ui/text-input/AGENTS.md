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

Only the props below reach the `<input>`; other HTML attributes (`id`, `onBlur`, `aria-*`, …) are
not accepted.

| Prop | Type | Default | Description |
|---|---|---|---|
| `value` | `string` | — | Controlled value. Passing it (anything but `undefined`) makes the input controlled |
| `defaultValue` | `string` | `''` | Initial value when uncontrolled |
| `onValueChange` | `(value: string) => void` | — | Called with the new string on typing and on clear (`''`) |
| `placeholder` | `string` | — | |
| `type` | `'text' \| 'password'` | `'text'` | |
| `name` | `string` | — | |
| `icon` | `ReactNode` | — | Leading icon; its svg is forced to `size-4` |
| `suffix` | `ReactNode` | — | Trailing content, after the clear button; not sized or coloured by the component |
| `invalid` | `boolean` | — | Error styling; sets `aria-invalid` on the wrapper `<div>`, not on the `<input>` |
| `disabled` | `boolean` | — | Disables the `<input>` and hides the clear button |
| `font` | `'mono' \| 'sans'` | `'mono'` | Font of the typed text |
| `className` | `string` | — | Merged into the wrapper `<div>` classes |
| `ref` | `Ref<HTMLInputElement>` | — | Forwarded to the `<input>` |

The clear button (`aria-label="Clear"`, `tabIndex={-1}`) shows whenever the value is non-empty and
the input is not disabled; it sets the value to `''` and refocuses the input.

## Semantic styles

No cva variants. The wrapper is `h-9 w-full`, so the width comes from the parent or `className`.

| Part | Tokens |
|---|---|
| wrapper | `border-hairline`, `bg-canvas`, `rounded-lg`, `text-base` |
| typed text | `text-ink`; `font-mono` or `font-sans` by `font` |
| placeholder | `text-ink-tertiary` |
| leading icon | `text-ink-subtle` |
| clear button | `text-ink-subtle`, hover `text-ink-muted`, focus-visible `ring-primary/40` |

States, all on the wrapper:

- focus-within: `ring-2 ring-primary` with `ring-offset-2 ring-offset-canvas`.
- invalid: `border-danger`, focus ring `ring-danger`.
- disabled: `opacity-50`, `pointer-events-none`.
