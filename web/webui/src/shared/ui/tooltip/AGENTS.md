# Tooltip

A short hint shown on hover or focus of its trigger, built on `@radix-ui/react-tooltip`. Pick
it for a label or an explanation that needs no interaction; for a menu use
`@/shared/ui/dropdown-menu`, for a panel use `@/shared/ui/sheet`.
Storybook: `Overlays/Tooltip`.

## Parts

| Part | Renders | Role |
|---|---|---|
| `TooltipProvider` | Tooltip `Provider` | shared open delay for the tooltips below it |
| `Tooltip` | Tooltip `Root` | one tooltip: `open` / `onOpenChange` / `defaultOpen` |
| `TooltipTrigger` | Tooltip `Trigger` | the element the hint belongs to; `asChild` to wrap an existing one |
| `TooltipContent` | Tooltip `Portal` + `Content` | the bubble, always rendered through a portal |

Every part forwards the props of its Radix primitive and sets `data-slot`.

## Usage

```tsx
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'

<Tooltip>
    <TooltipTrigger asChild>
        <Button variant="ghost">…</Button>
    </TooltipTrigger>
    <TooltipContent side="top">Tooltip label</TooltipContent>
</Tooltip>
```

`TooltipProvider` is already mounted: in `@/app` `Providers` for the app, in
`.storybook/preview.tsx` for stories, and inside `SidebarProvider`. Callers add none.

## Props

`TooltipContent` — Tooltip `Content` props (`side`, `align`, `hidden`, …) plus:

| Prop | Type | Default | Description |
|---|---|---|---|
| `variant` | `'default' \| 'primary'` | `'default'` | colour of the bubble |
| `sideOffset` | `number` | `4` | gap between trigger and bubble, px |

`TooltipProvider` — Tooltip `Provider` props, with one default changed:

| Prop | Type | Default | Description |
|---|---|---|---|
| `delayDuration` | `number` | `0` | delay before a tooltip opens, ms |

`Tooltip` and `TooltipTrigger` have no props of their own.

## Semantic styles

| `variant` | Tokens | Meaning |
|---|---|---|
| `default` | `bg-ink`, `text-canvas` | neutral hint, inverted against the page |
| `primary` | `bg-primary`, `text-on-primary` | emphasised hint; the operations list uses it for the error text |

Common to both: `rounded-md`, `text-xs`, `font-medium`, `select-none`, `shadow-md` (Tailwind
default, not a project token). The bubble fades and zooms in, sliding 1 unit away from the
trigger according to Radix `data-side`; `data-state=closed` reverses it.
