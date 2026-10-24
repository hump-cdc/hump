# Tooltip

A short hint shown on hover or focus of its trigger, built on `@radix-ui/react-tooltip`. Pick
it for a label or an explanation that needs no interaction; for a menu use
`@/shared/ui/dropdown-menu`, for a panel use `@/shared/ui/sheet`.
Storybook: `Overlays/Tooltip`.

## Parts

One part per Radix primitive: `TooltipProvider`, `Tooltip` (`Root`), `TooltipTrigger`,
`TooltipContent`. `TooltipContent` always renders through a portal.

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

- `TooltipContent`: `variant` (`'default'` or `'primary'`) and `sideOffset` (default `4` px), plus
  the Radix `Content` props.
- `TooltipProvider`: `delayDuration` defaults to `0` (Radix's own default is not used).

## Semantic styles

| `variant` | Tokens | Meaning |
|---|---|---|
| `default` | `bg-ink`, `text-canvas` | neutral hint, inverted against the page |
| `primary` | `bg-primary`, `text-on-primary` | emphasised hint; the operations list uses it for the error text |

Both are `text-xs font-medium` with `shadow-md` (Tailwind default, not a project token). The
bubble fades and zooms in, sliding away from the trigger by Radix `data-side`.
