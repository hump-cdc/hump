# Tabs

Switches between views or between a few short options, built on `@radix-ui/react-tabs`, with an
indicator that slides to the active trigger. `pill` is a compact segmented control, `underline`
is a full-width tab strip above panels.
Storybook: `Layout/Tabs`.

## Parts

| Part | Renders | Role |
|---|---|---|
| `Tabs` | Tabs `Root` | state holder: `value` / `onValueChange` / `defaultValue` |
| `TabsList` | Tabs `List` | the row of triggers; owns the variant and the indicator |
| `TabsTrigger` | Tabs `Trigger` | one tab (`value`, `disabled`) |
| `TabsContent` | Tabs `Content` | the panel for one `value`; optional |

Every part forwards the props of its Radix primitive and sets `data-slot`. The indicator
(`tabs-list-indicator.tsx`) is internal and not exported.

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

`TabsList` — Tabs `List` props plus:

| Prop | Type | Default | Description |
|---|---|---|---|
| `variant` | `'pill' \| 'underline'` | `'pill'` | shape of the list and of the active indicator |

`Tabs`, `TabsTrigger` and `TabsContent` have no props of their own.

## Semantic styles

`variant` is set on `TabsList` only. It is mirrored to `data-variant`, and `TabsTrigger` restyles
itself through `group-data-[variant=underline]/tabs-list:`.

| | `pill` | `underline` |
|---|---|---|
| list | `inline-flex w-fit`, `rounded-lg`, `border-hairline`, `bg-surface-1`, `p-0.5` | `flex w-full`, `border-hairline border-b` |
| trigger | `rounded-md px-3 py-1.5 text-sm`, width by content | `flex-1` (equal widths), `rounded-none`, `text-base` |
| active trigger | `text-ink` | `text-ink`, `font-semibold` |
| indicator | covers the trigger: `rounded-md`, `border-hairline-strong`, `bg-canvas` (`dark:bg-surface-3`), `shadow-sm` | 2px bar on the trigger's bottom edge, `bg-primary` |

Trigger states, both variants:

| State | Classes |
|---|---|
| inactive | `text-ink-subtle`, `font-medium` |
| hover | `text-ink-muted` |
| active (`data-state=active`) | `text-ink` |
| disabled | `opacity-50`, `pointer-events-none` |
| focus-visible | `ring-2 ring-primary`, `ring-offset-2 ring-offset-canvas` |

Icons inside a trigger default to `size-4` unless they carry their own `size-*` class.

The indicator is an absolutely positioned `div` inside the list (the list is `relative`). It
finds `[data-slot="tabs-trigger"][data-state="active"]`, copies that trigger's
`offsetLeft` / `offsetTop` / `offsetWidth` / `offsetHeight`, and re-measures on `data-state`
changes, on added or removed triggers (`MutationObserver`) and on list resize (`ResizeObserver`).
With no active trigger it is hidden (`opacity: 0`). The slide transition (300ms) switches on
only after the first measurement, so the indicator does not animate in on mount.

`Tabs` itself is `flex flex-col gap-2`; `TabsContent` is `flex-1 outline-none`.
