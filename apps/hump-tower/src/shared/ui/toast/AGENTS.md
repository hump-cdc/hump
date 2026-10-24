# Toast

Transient notifications over the page, built on `sonner`: the `Toaster` host styled with the
kit's tokens, and sonner's `toast` function re-exported unchanged. Use it for the outcome of an
action; for a state that stays on the page use `Placeholder` (`placeholder`).
Storybook: `Overlays/Toast`.

`Toaster` is mounted once, in `src/app/providers.tsx`; without it `toast()` shows nothing.

## Usage

```tsx
import { toast } from '@/shared/ui/toast'

toast.error('Failed to start the snapshot', { description: errorMessage(error) })

toast('Sink reconnected', {
    description: 'Delivery resumed from the last confirmed position.',
    action: { label: 'Undo', onClick: handleUndo },
})
```

## Props

`Toaster` takes sonner's `Toaster` props except `closeButton`. `position` defaults to
`'bottom-center'`; `style` is merged over the token variables; `toastOptions.classNames` are merged
over the kit's own, per key. `toast` takes sonner's arguments unchanged.

## Semantic styles

The cva `variant` is internal: the toast type picks it.

| `toast` call | Variant | Background | Border |
|---|---|---|---|
| `toast()`, `toast.info()`, `toast.loading()` | `default` | `--surface-1` | `--hairline-strong` |
| `toast.success()` | `success` | `--success-soft` | `--success` at 30% |
| `toast.error()` | `danger` | `--danger-soft` | `--danger` at 30% |
| `toast.warning()` | `amber` | `--amber-soft` | `--amber` at 30% |

- Description `text-ink-subtle`; action button `bg-primary text-on-primary`, cancel button
  `bg-surface-3 text-ink-muted`.
- Icons: only `loading` has one (a spinning `LoaderCircle`); the success, info, warning and error
  icons are removed, so the type shows through colour alone.
