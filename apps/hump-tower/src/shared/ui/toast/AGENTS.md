# Toast

Transient notifications over the page, built on `sonner`: the `Toaster` host styled with the
kit's tokens, and sonner's `toast` function re-exported unchanged. Use it for the outcome of an
action; for a state that stays on the page use `Placeholder` (`placeholder`).
Storybook: `Overlays/Toast`.

Parts (`index.ts`):

- `Toaster` — the host that renders the toasts. Mounted once, in `src/app/providers.tsx`;
  without it `toast()` shows nothing.
- `toast` — sonner's API: `toast()`, `toast.success()`, `toast.error()`, `toast.warning()`,
  `toast.info()`, `toast.loading()`.

## Usage

```tsx
import { Toaster, toast } from '@/shared/ui/toast'

// once, at the app root
<Toaster />

// from any slice
toast.error('Failed to start the snapshot', { description: errorMessage(error) })

toast('Sink reconnected', {
    description: 'Delivery resumed from the last confirmed position.',
    action: { label: 'Undo', onClick: handleUndo },
})
```

## Props

`Toaster` — the props of sonner's `Toaster`, except `closeButton`.

| Prop | Type | Default | Description |
|---|---|---|---|
| `position` | sonner's `position` | `'bottom-center'` | where the toasts stack |
| `style` | sonner's `style` | — | merged over the token variables below |
| `toastOptions` | sonner's `toastOptions` | — | `classNames` are merged over the kit's own, per key |

`toast` takes sonner's arguments; nothing is added or removed.

## Semantic styles

The cva `variant` is internal: the toast type picks it.

| `toast` call | Variant | Background | Border |
|---|---|---|---|
| `toast()`, `toast.info()`, `toast.loading()` | `default` | `--surface-1` | `--hairline-strong` |
| `toast.success()` | `success` | `--success-soft` | `--success` at 30% |
| `toast.error()` | `danger` | `--danger-soft` | `--danger` at 30% |
| `toast.warning()` | `amber` | `--amber-soft` | `--amber` at 30% |

- Text `--ink`; title `font-medium text-ink`; description `text-ink-subtle`.
- Action button `bg-primary text-on-primary`, hover `bg-primary-hover`; cancel button
  `bg-surface-3 text-ink-muted`.
- Icons: only `loading` has one (a spinning `LoaderCircle`, `text-ink-subtle`); the success,
  info, warning and error icons are removed, so the type shows through colour alone.
- Hover background `--surface-2`; radius `0.5rem`.
