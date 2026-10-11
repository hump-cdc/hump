# Sheet

A panel that slides in from one edge, built on `@radix-ui/react-dialog`. Pick it for a side
drawer or a details panel next to the page content; for a small anchored menu use
`@/shared/ui/dropdown-menu`, for a hint use `@/shared/ui/tooltip`.
Storybook: `Overlays/Sheet`.

## Parts

| Part | Renders | Role |
|---|---|---|
| `Sheet` | Dialog `Root` | state holder: `open` / `onOpenChange` / `defaultOpen` / `modal` |
| `SheetTrigger` | Dialog `Trigger` | opens the sheet; `asChild` to wrap a `Button` |
| `SheetContent` | Dialog `Portal` + `Overlay` + `Content` | the panel itself, plus overlay and close button |
| `SheetHeader` | `div` | padded column for title and description |
| `SheetTitle` | Dialog `Title` | accessible name of the panel |
| `SheetDescription` | Dialog `Description` | accessible description |
| `SheetClose` | Dialog `Close` | closes the sheet; `asChild` to wrap a custom button |
| `SheetOverlay` | Dialog `Overlay` | the backdrop; `SheetContent` already renders it |

Every part forwards the props of its Radix primitive (or of `div`) and sets `data-slot`. The
portal is not exported: `SheetContent` owns it.

## Usage

```tsx
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from '@/shared/ui/sheet'

<Sheet>
    <SheetTrigger asChild>
        <Button variant="outline">Filters</Button>
    </SheetTrigger>
    <SheetContent side="right">
        <SheetHeader>
            <SheetTitle>Filters</SheetTitle>
            <SheetDescription>Narrow the list.</SheetDescription>
        </SheetHeader>
        <div className="flex flex-col gap-3 p-4">…</div>
    </SheetContent>
</Sheet>
```

Controlled, without a trigger: `<Sheet open={open} onOpenChange={setOpen}>`
(`toolbar/toolbar.tsx`, `sidebar/sidebar.tsx`).

Docked, non-modal panel that leaves the page usable — the operation details and the flow
inspector:

```tsx
<Sheet modal={false} open={Boolean(selected)} onOpenChange={(open) => !open && clear()}>
    <SheetContent
        side="right"
        portal={false}
        showOverlay={false}
        showCloseButton={false}
        aria-describedby={undefined}
        onOpenAutoFocus={(event) => event.preventDefault()}
        onInteractOutside={(event) => event.preventDefault()}
        onPointerDownOutside={(event) => event.preventDefault()}
        className="absolute z-10 gap-0 bg-surface-1 p-0"
    >
        <SheetHeader>
            <SheetTitle>…</SheetTitle>
            <SheetClose asChild>…</SheetClose>
        </SheetHeader>
    </SheetContent>
</Sheet>
```

- `portal={false}` renders the panel inline; `SheetContent` is `fixed`, so the caller overrides
  the position (`absolute`) to dock it inside a positioned ancestor.
- Without `SheetDescription` the callers pass `aria-describedby={undefined}`. A sheet with no
  visible heading keeps `SheetTitle` inside an `sr-only` `SheetHeader` (`sidebar/sidebar.tsx`).

## Props

`SheetContent` — Dialog `Content` props plus:

| Prop | Type | Default | Description |
|---|---|---|---|
| `side` | `'right' \| 'left' \| 'top' \| 'bottom'` | `'right'` | edge the panel is anchored to and slides from |
| `showCloseButton` | `boolean` | `true` | renders the `X` button in the top right corner |
| `showOverlay` | `boolean` | `true` | renders `SheetOverlay` behind the panel |
| `portal` | `boolean` | `true` | `true` portals to `document.body`; `false` renders inline |

The other parts have no props of their own.

## Semantic styles

`side` is the only cva variant (`SheetContent`):

| `side` | Geometry | Border |
|---|---|---|
| `right` | full height, `w-3/4`, `sm:max-w-sm` | `border-l` |
| `left` | full height, `w-3/4`, `sm:max-w-sm` | `border-r` |
| `top` | full width, height by content | `border-b` |
| `bottom` | full width, height by content | `border-t` |

| Part | Tokens |
|---|---|
| `SheetContent` | `bg-canvas`, `text-ink`, `border-hairline`; `shadow-xl` (Tailwind default, not a project token) |
| close button | `text-ink-subtle` → `hover:text-ink`; focus ring `ring-primary` |
| `SheetOverlay` | `bg-black/50` — a raw colour, there is no overlay token |
| `SheetTitle` | `text-ink`, `font-semibold` |
| `SheetDescription` | `text-ink-muted`, `text-sm` |

States come from Radix `data-state`: `open` slides in from the side over 300ms, `closed`
slides out over 200ms; the overlay fades. `className` on `SheetContent` is merged after the
variant classes, so it can override width, background and padding.
