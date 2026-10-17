# MultiComboBox

A ready-made multi-select with search: selected values as chips on one line, a popup of checkbox
options, "Select all" and a counter. Use it for a filter over a known list
(`entities/flows/toolbar`, `entities/operations/filter`); drop down to the parts in
`@/shared/ui/combobox` only for a combobox this one does not cover.
Storybook: `Inputs/Multi Combo Box`.

Exports: `MultiComboBox` and the type `MultiComboBoxItem`.

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

Only the props below; nothing else is passed through to the combobox.

| Prop | Type | Default | Description |
|---|---|---|---|
| `items` | `MultiComboBoxItem[]` | required | The options |
| `value` | `string[]` | — | Selected `value`s. Passing it makes the component controlled; without it the selection is internal and starts empty |
| `onValueChange` | `(value: string[]) => void` | — | Called with the new selected `value`s |
| `placeholder` | `string` | — | Shown in the input only while nothing is selected |
| `emptyText` | `string` | `'No items found.'` | Popup text when the search matches nothing |
| `icon` | `ReactNode` | — | Leading icon; its svg is forced to `size-4` |
| `hint` | `ReactNode` | — | Tooltip on hovering the icon; ignored without `icon` |
| `invalid` | `boolean` | — | Error styling; sets `aria-invalid` on the field box |
| `disabled` | `boolean` | — | |
| `clearToAll` | `boolean` | — | The clear button selects every item instead of none, for a filter where "All" is the reset state |
| `className` | `string` | — | Passed to the field box (`ComboboxChips`); the place to set the width |
| `ref` | `Ref<HTMLDivElement>` | — | Forwarded to the field box, which is also the popup anchor |

`MultiComboBoxItem`:

| Field | Type | Description |
|---|---|---|
| `value` | `string` | Identity of the option; what `value` / `onValueChange` carry |
| `label` | `string` | Text of the option and its chip; what the search matches |
| `aside` | `string` (optional) | Secondary text on the right of the option, e.g. a count |

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
`@/shared/ui/combobox` (including focus, `aria-invalid` and disabled states). Own pieces:

| Part | Tokens |
|---|---|
| leading icon | `text-ink-subtle` |
| `All` badge | `bg-surface-3`, `text-ink-muted` — neutral: nothing is filtered out |
| `+N` badge | `bg-primary-soft`, `text-primary` — same colours as a chip |
| chevron (`ChevronsUpDown`) | `text-ink-subtle` |
| option checkbox | `border-hairline-strong`; selected `bg-primary` with the `Check` in `text-on-primary` |
| option label / `aside` | `text-ink` / `text-ink-tertiary`; `aside` also `font-mono`, `tabular-nums` |
| footer | `border-hairline` top border; "Select all" `text-primary`, hover `text-primary-hover`, disabled `opacity-50` once everything is selected; counter `text-ink-tertiary` |
