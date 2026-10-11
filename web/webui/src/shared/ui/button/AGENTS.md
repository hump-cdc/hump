# Button

The kit's text/icon button: the default for any action the user clicks. For the icon-only
copy-to-clipboard control use `CopyButton` (`@/shared/ui/copy-button`) instead.
Storybook: `Actions/Button`.

## Usage

```tsx
import { Plus } from 'lucide-react'

import { Button } from '@/shared/ui/button'

<Button variant="outline" loading={isPending} onClick={onAdd}>
    <Plus />
    Add
</Button>
```

As a link (`pages/not-found`): the child becomes the rendered element and takes the classes.

```tsx
<Button asChild>
    <Link to="/" replace>
        Go to Flows
    </Link>
</Button>
```

## Props

Plus every `<button>` attribute (`ButtonHTMLAttributes<HTMLButtonElement>`), passed through.

| Prop | Type | Default | Description |
|---|---|---|---|
| `variant` | `'primary' \| 'secondary' \| 'outline' \| 'ghost' \| 'danger'` | `'primary'` | Visual weight, see below |
| `nonInteractive` | `boolean` | `false` | `true` drops the press (scale) feedback |
| `loading` | `boolean` | `false` | Shows a spinning `LoaderCircle` before the children, hides every other svg inside, and disables the button |
| `disabled` | `boolean` | — | Effective value is `disabled \|\| loading` |
| `asChild` | `boolean` | `false` | Renders the single child through Radix `Slot` instead of a `<button>` |
| `className` | `string` | — | Merged through `cn` after the variant classes |
| `ref` | `Ref<HTMLButtonElement>` | — | Forwarded to the element |

With `asChild` the children are rendered as they are: no spinner is injected, though `loading`
still hides the child's svgs and passes `disabled` to the child.

## Semantic styles

Base: `h-8.5`, `rounded-lg`, `px-3.5`, `gap-2`, `font-medium text-base`; any svg inside is forced to
`size-4`.

| `variant` | Meaning | Tokens |
|---|---|---|
| `primary` | the main action of a block | `bg-primary`, `text-on-primary`, hover `bg-primary-hover` |
| `secondary` | a filled, lower-emphasis action | `bg-surface-3`, `text-ink-muted`, hover `bg-surface-2` |
| `outline` | a bordered action on a plain background | `border-hairline-strong`, `text-ink-muted`, hover `bg-surface-1` |
| `ghost` | the quietest action, no fill or border | `text-ink-subtle`, hover `bg-surface-1` + `text-ink-muted` |
| `danger` | a destructive action | `bg-danger`, `text-white`, hover `opacity-90` |

| `nonInteractive` | Classes |
|---|---|
| `false` | `active:scale-[0.96]`, cancelled under `motion-reduce` |
| `true` | none |

States:

- focus-visible: `ring-2 ring-primary` with `ring-offset-2 ring-offset-canvas`.
- disabled (or `loading`): `opacity-50`, `pointer-events-none`.
