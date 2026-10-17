# Table

A plain flex-based list table for a short, fully loaded set of rows, with a responsive layout
(one line on `lg` and up, stacked zones below) and optional checkbox rows. For a long or lazily
loaded list that needs virtualization, column resizing or expandable rows pick
`@/shared/ui/data-table`.
Storybook: `Content/Table`.

Parts (`index.ts`):

| Export | Kind | Role |
|---|---|---|
| `Table` | component | frame, optional header and toolbar, the column header row, then `children` |
| `TableRow` | component | one row; reads the variant from the enclosing `Table` |
| `TableRowHeader` | component | the column titles; `Table` renders it itself |
| `TableRowColumn<T>`, `MobileZone` | types | column definition |
| `TableVariant` | type | `'default' \| 'bordered'` |
| `SelectableProps` | type | the `selectable` object of `TableRow` |

## Usage

```tsx
import { Table, TableRow, type TableRowColumn } from '@/shared/ui/table'

const COLUMNS: TableRowColumn<Item>[] = [
    {
        key: 'name',
        header: 'Name',
        className: 'flex-1 truncate',
        cell: (item) => <span className="font-mono text-base text-ink">{item.name}</span>,
        mobile: { zone: 'title', render: (item) => <span className="truncate">{item.name}</span> },
    },
]

<Table columns={COLUMNS} variant="bordered" header="Items" headerAside={<span>3 total</span>}>
    {items.map((item) => (
        <TableRow key={item.id} row={item} columns={COLUMNS} />
    ))}
</Table>
```

The same `columns` array goes to `Table` (for the header row) and to every `TableRow`. Column
widths are Tailwind classes in `className` (`flex-1`, `w-32`), shared by the header cell and the
body cell. A `TableRow` outside a `Table` falls back to the `default` variant.

Selectable rows — selection state is the caller's:

```tsx
<Table columns={COLUMNS} selectable>
    {items.map((item) => (
        <TableRow
            key={item.id}
            row={item}
            columns={COLUMNS}
            selectable={{ id: `item-${item.id}`, checked: checked.has(item.id), onToggle: () => toggle(item.id) }}
        />
    ))}
</Table>
```

## Props

### Table

| Prop | Type | Default | Description |
|---|---|---|---|
| `columns` | `TableRowColumn<T>[]` | required | Columns of the header row |
| `variant` | `'default' \| 'bordered'` | `'default'` | `default` — no frame; `bordered` — wrapped in a `Card` |
| `selectable` | `boolean` | — | Reserves the checkbox slot in the header row |
| `header` | `ReactNode` | — | Title area above the table |
| `headerAside` | `ReactNode` | — | Right side of the title area |
| `toolbar` | `ReactNode` | — | Strip between the title area and the column header |
| `children` | `ReactNode` | required | The `TableRow`s |
| `className` | `string` | — | On the `Card` (`bordered`) or the wrapper `div` (`default`) |

The title area is rendered only when `header` or `headerAside` is set.

### TableRow

| Prop | Type | Default | Description |
|---|---|---|---|
| `row` | `T` | required | Row data handed to the column renderers |
| `columns` | `TableRowColumn<T>[]` | required | |
| `selectable` | `{ id: string; checked: boolean; onToggle: () => void }` | — | Makes the row a `<label>` with a `Checkbox`; `id` must be unique on the page |
| `mobileZoneClasses` | `Partial<Record<MobileZone, string>>` | — | Extra classes per mobile zone |

### TableRowHeader

| Prop | Type | Default | Description |
|---|---|---|---|
| `columns` | `TableRowColumn<T>[]` | required | |
| `selectable` | `boolean` | — | Adds the checkbox-sized spacer |

### TableRowColumn<T>

| Field | Type | Description |
|---|---|---|
| `key` | `string` | Stable id, used as the React key |
| `header` | `string` | Column title; an empty cell with `className` is rendered when omitted |
| `className` | `string` | Width / alignment of the header cell and the `lg` body cell |
| `cell` | `(row: T) => ReactNode` | Body cell on `lg` and up |
| `mobile` | `{ zone: MobileZone; className?: string; render: (row: T) => ReactNode }` | Below `lg`: which zone the column goes to and what it renders |

## Layout

- `lg` and up: one flex line per row, `gap-4`, each `cell` wrapped in a `div` with the column's
  `className`. The column header row is `hidden` below `lg`.
- Below `lg`, plain rows: zones stacked in the fixed order `title`, `meta`, `fill`, `badges`. A
  column without `mobile` is not rendered at all; an empty zone is skipped.
- Selectable rows render `cell` for every column at every width — `mobile` is not used, and the
  cells are not wrapped in the column's `className`, so `cell` has to carry its own width.

## Semantic styles

Every part follows the `Table` `variant`, passed down through context. The default title area,
column header and row are `cva` variants; frame and toolbar are ternaries in `table.tsx`:

| Part | `default` | `bordered` |
|---|---|---|
| frame | `div`, `text-ink` | `Card` (`bg-surface-1`, `border-hairline`, `rounded-xl`) with `overflow-visible` |
| title area | `pb-2` | `CardHeader` |
| toolbar | `py-4`, `border-hairline` bottom | `p-4`, `border-hairline` bottom |
| column header | `py-2.5` | `py-2.5`, plus `bg-surface-2 px-4` |
| row | `py-3` | `py-3 px-4`, last row `rounded-b-xl` |

Column titles are `text-ink-subtle text-xs font-semibold uppercase tracking-wide`. Rows are
separated by `border-hairline`, none after the last. Mobile zone `meta` is
`font-mono text-sm text-ink-subtle`.

States, selectable rows only: hover `bg-surface-2`, checked `bg-primary-soft/40`,
`cursor-pointer`. Plain rows have no hover state.
