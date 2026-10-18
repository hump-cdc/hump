# Toolbar

The filter bar above a page's content: a `Card` holding one row of controls on desktop, which
collapses into a single button opening a `Sheet` on mobile. Used by `pages/operations` and
`entities/flows/toolbar`.
Storybook: `Layout/Toolbar`.

## Usage

```tsx
import { Search } from 'lucide-react'

import { MultiComboBox } from '@/shared/ui/multi-combo-box'
import { TextInput } from '@/shared/ui/text-input'
import { Toolbar } from '@/shared/ui/toolbar'

<Toolbar label="Filters">
    <MultiComboBox items={statusItems} placeholder="Status" className="w-72" />
    <TextInput placeholder="Search…" icon={<Search />} font="sans" className="w-full" />
</Toolbar>
```

Layout by `useIsMobile()` from `@/shared/utils` (below 1024px):

```
desktop                           mobile
Card                              Card
    children  actions                 Button (outline, SlidersHorizontal + label) → Sheet (right)
                                          SheetTitle: label
                                          children, stacked
                                      actions
```

## Props

| Prop | Type | Default | Description |
|---|---|---|---|
| `children` | `ReactNode` | required | The controls; one row on desktop, inside the `Sheet` on mobile |
| `actions` | `ReactNode` | — | Controls kept out of the mobile `Sheet`: below the trigger on mobile, trailing the row on desktop |
| `label` | `string` | `'Filters'` | Mobile trigger label and `Sheet` title |
| `className` | `string` | — | Merged into the `CardContent` classes, not the `Card` |
| `ref` | `Ref<HTMLDivElement>` | — | Forwarded to the `Card` |

The children must be direct elements, not wrapped in one container: on mobile every direct child
gets `w-full` (`*:w-full`), both the stacked controls in the `Sheet` and the trigger and `actions`
below it. Widths set by the caller (`w-72`, `w-1/2`) apply on desktop only.

The `Sheet`'s open state is internal; there is no prop to control it.

## Semantic styles

No cva variants; the surface comes from `Card` and `CardContent` (`@/shared/ui/card`), the mobile
trigger from `Button variant="outline"`, the panel from `Sheet side="right"` (`@/shared/ui/sheet`).

| Part | Classes |
|---|---|
| `CardContent` | `px-4 pt-2.5 pb-3` |
| desktop row | `flex items-center gap-2` |
| mobile column | `flex flex-col gap-2` |
| controls inside the `Sheet` | `flex flex-col gap-3 p-4` |
