# MultiComboBox

A ready-made multi-select with search: selected values as chips on one line, a popup of checkbox
options, "Select all" and a counter. Use it for a filter over a known list
(`entities/flows/toolbar`, `entities/operations/filter`); drop down to the parts in
`@/shared/ui/combobox` only for a combobox this one does not cover.
Storybook: `Inputs/Multi Combo Box`.

Exports: `MultiComboBox` and the type `MultiComboBoxItem` (`{ value, label, aside? }`).

## Usage

```tsx
import { Table2 } from 'lucide-react'

import { MultiComboBox, type MultiComboBoxItem } from '@/shared/ui/multi-combo-box'

const tables: MultiComboBoxItem[] = [
    { value: 'public.orders', label: 'orders', aside: '1,024' },
    { value: 'public.users', label: 'users' },
]

<MultiComboBox
    clearToAll
    items={tables}
    value={selected}
    onValueChange={setSelected}
    placeholder="Tables"
    icon={<Table2 />}
    hint="Tables to show on the flow, with their connections"
    className="w-1/2"
/>
```

## Props

The props type in `multi-combo-box.tsx` is closed: nothing else is passed through to the combobox.
Not obvious from the types:

- `value` makes it controlled; without it the selection is internal and starts empty.
- `label` is what the option, its chip and the search show; `aside` is secondary text on the
  option's right (e.g. a count).
- `placeholder` shows only while nothing is selected.
- `hint` is a tooltip on the icon and is ignored without `icon`.
- `clearToAll` makes the clear button select every item instead of none, for a filter where "All"
  is the reset state.
- `className` and `ref` go to the field box (`ComboboxChips`), which is also the popup anchor:
  set the width there.

Behaviour a caller should know:

- Entries of `value` that match no item are dropped silently; the chips follow the order of
  `value`.
- Chips never wrap: those that do not fit collapse into a `+N` badge, recomputed on resize.
- When every item is selected, one neutral `All` badge replaces the chips.
- The clear button shows while something is selected; with `clearToAll`, while not everything is.
- "Select all" in the popup footer selects every item, not only the ones matching the search.
- `hint` renders a `Tooltip` (`@/shared/ui/tooltip`), which needs `TooltipProvider` above — mounted
  in `@/app` `Providers`.

## Semantic styles

No cva variants. The field box, chips, popup and option rows take their styles from
`@/shared/ui/combobox` (including focus, `aria-invalid` and disabled states). Own pieces: the `All`
badge is neutral (`bg-surface-3 text-ink-muted`: nothing is filtered out), the `+N` badge uses the
chip colours (`bg-primary-soft text-primary`), a selected option's checkbox is `bg-primary`, and
"Select all" is `text-primary`, dimmed once everything is selected.
