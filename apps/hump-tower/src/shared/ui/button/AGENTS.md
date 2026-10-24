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

| Prop | Description |
|---|---|
| `nonInteractive` | `true` drops the press (scale) feedback |
| `loading` | Shows a spinning `LoaderCircle` before the children, hides every other svg inside, and disables the button (`disabled \|\| loading`) |

With `asChild` the children are rendered as they are: no spinner is injected, though `loading`
still hides the child's svgs and passes `disabled` to the child.

## Semantic styles

- `primary` — the main action of a block; `secondary` — a filled, lower-emphasis action;
  `outline` — a bordered action on a plain background; `ghost` — the quietest action, no fill or
  border; `danger` — a destructive action.
- `danger` uses `text-white` and hovers with `opacity-90` instead of a hover token.
- The press scale (`active:scale-[0.96]`, unless `nonInteractive`) is cancelled under
  `motion-reduce`. Disabled and `loading` look the same: `opacity-50`, `pointer-events-none`.
