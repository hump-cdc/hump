# Tabs

Switches between views or between a few short options, built on `@radix-ui/react-tabs`, with an
indicator that slides to the active trigger. `pill` is a compact segmented control, `underline`
is a full-width tab strip above panels.
Storybook: `Layout/Tabs`.

## Parts

One part per Radix primitive: `Tabs` (`Root`), `TabsList`, `TabsTrigger`, `TabsContent` (optional).
`TabsList` owns the variant and the indicator; the indicator (`tabs-list-indicator.tsx`) is internal.

## Usage

Tab strip with panels, uncontrolled (`entities/operations/details/ui/operation-details.tabs.tsx`):

```tsx
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'

<Tabs defaultValue="event">
    <TabsList variant="underline">
        <TabsTrigger value="event">Event</TabsTrigger>
        <TabsTrigger value="payload">Payload</TabsTrigger>
    </TabsList>
    <TabsContent value="event">…</TabsContent>
    <TabsContent value="payload">…</TabsContent>
</Tabs>
```

Segmented control without panels, controlled (`entities/config/state/ui/live-controls.tsx`):

```tsx
<Tabs value={chartStep} onValueChange={(value) => setChartStep(value as ChartStep)}>
    <TabsList className="w-full">
        {CHART_STEPS.map((option) => (
            <TabsTrigger key={option} value={option} className="flex-1 py-1 text-xs">
                {option}
            </TabsTrigger>
        ))}
    </TabsList>
</Tabs>
```

`onValueChange` hands back a `string`; the caller narrows it.

## Props

Only `TabsList` adds a prop: `variant` (`'pill'` default, or `'underline'`). The other parts take
their Radix props.

## Semantic styles

`variant` is set on `TabsList` only. It is mirrored to `data-variant`, and `TabsTrigger` restyles
itself through `group-data-[variant=underline]/tabs-list:`.

| | `pill` | `underline` |
|---|---|---|
| list | `w-fit`, bordered `bg-surface-1` box | `w-full`, `border-b` |
| trigger | width by content | equal widths (`flex-1`), active one `font-semibold` |
| indicator | covers the trigger: `bg-canvas` (`dark:bg-surface-3`), `border-hairline-strong` | 2px `bg-primary` bar on the trigger's bottom edge |

Triggers in both variants: inactive `text-ink-subtle`, hover `text-ink-muted`, active `text-ink`,
disabled `opacity-50`. Icons inside a trigger default to `size-4` unless they carry their own
`size-*` class.

The indicator is an absolutely positioned `div` inside the list (the list is `relative`). It
finds `[data-slot="tabs-trigger"][data-state="active"]`, copies that trigger's offset box, and
re-measures on `data-state` changes, on added or removed triggers (`MutationObserver`) and on list
resize (`ResizeObserver`). With no active trigger it is hidden (`opacity: 0`). The slide transition
(300ms) switches on only after the first measurement, so the indicator does not animate in on
mount.
