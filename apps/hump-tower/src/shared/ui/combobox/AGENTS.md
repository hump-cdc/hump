# Combobox

Styled parts of the Base UI combobox (`@base-ui/react/combobox`, not Radix): a text field that
filters a popup list. These are building blocks. For multi-select use `MultiComboBox`
(`@/shared/ui/multi-combo-box`), which is assembled from them and is their only consumer today;
compose the parts directly only for a combobox it does not cover, e.g. single-select.
Storybook: `Inputs/Combobox`.

## Parts

| Export | Base UI part | Role |
|---|---|---|
| `Combobox` | `Root` | state: `items`, `value`, `onValueChange`, `multiple`, `disabled`, … Re-exported unstyled |
| `ComboboxValue` | `Value` | render prop over the current value. Re-exported unstyled |
| `ComboboxChips` | `Chips` | the field box; also holds the icon, input, clear button and chevron |
| `ComboboxChip` | `Chip` + `ChipRemove` | one selected value with its remove button |
| `ComboboxChipsInput` | `Input` | the text input inside the field box |
| `ComboboxClear` | `Clear` | the clear button (`X`, `aria-label="Clear selection"`) |
| `ComboboxContent` | `Portal` + `Positioner` + `Popup` | the popup |
| `ComboboxList` | `List` | the scrollable list; takes a render function per item |
| `ComboboxItem` | `Item` | one option |
| `ComboboxEmpty` | `Empty` | the "nothing found" text |
| `useComboboxAnchor` | — | `useRef<HTMLDivElement \| null>(null)` for the popup anchor |
| `useComboboxFilteredItems` | `useFilteredItems` | re-export of the Base UI hook |

## Usage

Single-select, as in the story:

```tsx
import { Check, ChevronsUpDown } from 'lucide-react'

import {
    Combobox,
    ComboboxChips,
    ComboboxChipsInput,
    ComboboxClear,
    ComboboxContent,
    ComboboxEmpty,
    ComboboxItem,
    ComboboxList,
    useComboboxAnchor,
} from '@/shared/ui/combobox'

const anchor = useComboboxAnchor()

<Combobox
    items={sinks}
    value={value}
    onValueChange={setValue}
    itemToStringLabel={(item: Sink) => item.label}
    isItemEqualToValue={(a: Sink, b: Sink) => a.value === b.value}
    disabled={disabled}
>
    <ComboboxChips ref={anchor} aria-invalid={invalid || undefined}>
        <ComboboxChipsInput placeholder="Select a sink…" />
        {value && <ComboboxClear disabled={disabled} />}
        <ChevronsUpDown className="pointer-events-none size-4 shrink-0 text-ink-subtle" />
    </ComboboxChips>

    <ComboboxContent anchor={anchor}>
        <ComboboxList>
            {(item: Sink) => (
                <ComboboxItem key={item.value} value={item}>
                    <span className="flex-1 truncate font-medium text-ink text-md">{item.label}</span>
                    <Check className="size-4 shrink-0 text-primary opacity-0 group-data-selected/combobox-item:opacity-100" />
                </ComboboxItem>
            )}
        </ComboboxList>
        <ComboboxEmpty>No sinks found.</ComboboxEmpty>
    </ComboboxContent>
</Combobox>
```

Constraints:

- The popup is as wide as its anchor (`w-(--anchor-width)`): put the same ref on `ComboboxChips`
  and in `ComboboxContent anchor`.
- `ComboboxEmpty` must sit inside `ComboboxContent`: it is `hidden` and shown only through the
  popup's `data-empty` (`group-data-empty/combobox-content:flex`).
- `ComboboxItem` renders no selection mark. Its children style themselves from
  `group-data-selected/combobox-item:`.
- There is no `invalid` prop: set `aria-invalid` on `ComboboxChips`.
- The chevron and the leading icon are plain children of `ComboboxChips`, not parts.

## Props

Every part takes the props of its Base UI part, passed through. Own props and changed defaults:

- `ComboboxChip` `showRemove` (default `true`): renders the remove button
  (`aria-label="Remove"`); `false` also evens the right padding.
- `ComboboxContent` takes the `Popup` props plus `side` (default `'bottom'`), `align`
  (`'start'`), `sideOffset` (`6`), `alignOffset` (`0`) and `anchor` from the `Positioner`.
  `className` and the remaining props go to the `Popup`, not the `Positioner`.

## Semantic styles

No cva variants. Each styled part sets a `data-slot` (`combobox-<part>`, plus
`combobox-chip-remove`).

- `ComboboxChips` has the same states as `TextInput`: focus-within ring, `aria-invalid` →
  `border-danger` with a `ring-danger` focus ring, disabled (`has-data-disabled`) →
  `opacity-50`.
- Selected chips are `bg-primary-soft` / `text-primary`; the highlighted item is `bg-surface-1`.
