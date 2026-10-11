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

Every part takes the props of its Base UI part, passed through, with `className` merged by `cn`.
Own props and changed defaults:

### ComboboxChip

| Prop | Type | Default | Description |
|---|---|---|---|
| `showRemove` | `boolean` | `true` | Renders the remove button (`aria-label="Remove"`); `false` also evens the right padding |

### ComboboxContent

Takes the `Popup` props plus five picked from the `Positioner`:

| Prop | Type | Default | Description |
|---|---|---|---|
| `side` | Base UI `Positioner` `side` | `'bottom'` | |
| `align` | Base UI `Positioner` `align` | `'start'` | |
| `sideOffset` | Base UI `Positioner` `sideOffset` | `6` | |
| `alignOffset` | Base UI `Positioner` `alignOffset` | `0` | |
| `anchor` | Base UI `Positioner` `anchor` | — | The element the popup is positioned against and sized to |

`className` and the remaining props go to the `Popup`, not the `Positioner`.

## Semantic styles

No cva variants. Each styled part sets a `data-slot` (`combobox-chips`, `combobox-chip`,
`combobox-chip-remove`, `combobox-chips-input`, `combobox-clear`, `combobox-content`,
`combobox-list`, `combobox-item`, `combobox-empty`).

| Part | Tokens |
|---|---|
| `ComboboxChips` | `h-9 w-full`, `border-hairline`, `bg-canvas`, `rounded-lg`; one line, overflow hidden |
| `ComboboxChip` | `bg-primary-soft`, `text-primary`, `h-5.5`, `rounded-md`; remove button `text-primary/70`, hover `text-primary` |
| `ComboboxChipsInput` | `text-ink`, placeholder `text-ink-tertiary` |
| `ComboboxClear` | `text-ink-subtle`, hover `text-ink-muted`, focus-visible `ring-primary/40` |
| `ComboboxContent` | `border-hairline`, `bg-canvas`, `rounded-xl`, `shadow-lg`, `z-50`; fade + zoom in and out |
| `ComboboxList` | `max-h-80`, scrolls vertically, `p-1.5` |
| `ComboboxItem` | `rounded-md`, `text-md`; highlighted `bg-surface-1`; disabled `opacity-50` |
| `ComboboxEmpty` | `text-ink-subtle`, centred, `py-6` |

States of the field box (`ComboboxChips`), the same as `TextInput`:

- focus-within: `ring-2 ring-primary` with `ring-offset-2 ring-offset-canvas`.
- `aria-invalid`: `border-danger`, focus ring `ring-danger`.
- disabled (`has-data-disabled`): `opacity-50`, `pointer-events-none`.
