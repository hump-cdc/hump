# Sheet

A panel that slides in from one edge, built on `@radix-ui/react-dialog`. Pick it for a side
drawer or a details panel next to the page content; for a small anchored menu use
`@/shared/ui/dropdown-menu`, for a hint use `@/shared/ui/tooltip`.
Storybook: `Overlays/Sheet`.

## Parts

One part per Dialog primitive: `Sheet` (`Root`), `SheetTrigger`, `SheetContent`, `SheetTitle`,
`SheetDescription`, `SheetClose`, `SheetOverlay`, plus `SheetHeader`, a padded `div` for title and
description. `SheetContent` renders the portal, the overlay and the close button itself; the
portal is not exported.

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

Controlled, without a trigger: `<Sheet open={open} onOpenChange={setOpen}>` (`sidebar/sidebar.tsx`).
`toolbar/toolbar.tsx` is controlled too but keeps its `<SheetTrigger asChild>`.

Docked, non-modal panel that leaves the page usable — the flow inspector
(`entities/flows/inspector`):

```tsx
<Sheet modal={false} open={Boolean(selected)} onOpenChange={(open) => !open && clear()}>
    <SheetContent
        side={isMobile ? 'bottom' : 'right'}
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
            <SheetClose aria-label="Close inspector">…</SheetClose>
        </SheetHeader>
    </SheetContent>
</Sheet>
```

- `portal={false}` renders the panel inline; `SheetContent` is `fixed`, so the caller overrides
  the position (`absolute`) to dock it inside a positioned ancestor.
- The operation details (`entities/operations/details`) are non-modal the same way but docked on
  desktop only: `portal={isMobile}`, and `absolute z-10` only on desktop, so the mobile bottom
  sheet portals to the viewport and spans it edge to edge.
- The flow inspector and the operation details have no `SheetDescription` and pass
  `aria-describedby={undefined}`; `toolbar/toolbar.tsx` has none either and passes nothing. A
  sheet with no visible heading keeps `SheetTitle` inside an `sr-only` `SheetHeader`
  (`sidebar/sidebar.tsx`).

## Props

`SheetContent` takes the Dialog `Content` props plus `side` (`'right'` default, `'left'`, `'top'`,
`'bottom'`), `showCloseButton` and `showOverlay` (both default `true`), and `portal` (default `true`:
portals to `document.body`; `false` renders inline). The other parts have no props of their own.

## Semantic styles

`side` is the only cva variant: `right` / `left` are full height, `w-3/4` up to `sm:max-w-sm`;
`top` / `bottom` are full width, height by content. The border sits on the inner edge.

The panel is `bg-canvas text-ink` with `border-hairline` and `shadow-xl` (Tailwind default, not a
project token). The overlay is `bg-black/50`, a raw colour: there is no overlay token.

States come from Radix `data-state`: `open` slides in over 300ms, `closed` slides out over 200ms;
the overlay fades. `className` on `SheetContent` is merged after the variant classes, so it can
override width, background and padding.
