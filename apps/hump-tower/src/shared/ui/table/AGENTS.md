# Table

A plain flex-based list table for a short, fully loaded set of rows, with a responsive layout
(one line on `lg` and up, stacked zones below) and optional checkbox rows. For a long or lazily
loaded list that needs virtualization, column resizing or expandable rows pick
`@/shared/ui/data-table`.
Storybook: `Content/Table`.

Exports: `Table`, `TableRow`, `TableRowHeader` (`Table` renders it itself), and the types
`TableRowColumn<T>`, `MobileZone`, `TableVariant`, `SelectableProps`.

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

Typed in `table.tsx`, `table-row.tsx` and `table.types.ts`. Not obvious from the types:

- `Table` `variant`: `default` has no frame, `bordered` wraps the table in a `Card`. `className`
  goes to the `Card` or to the wrapper `div`.
- The title area is rendered only when `header` or `headerAside` is set; `toolbar` sits between it
  and the column header.
- `Table` `selectable` only reserves the checkbox slot in the header row; each `TableRow` still
  needs its own `selectable` object. That object turns the row into a `<label>` around a
  `Checkbox` with `id`, so `id` must be unique on the page.
- A column without `header` gets an empty header cell that still carries its `className`.
- `mobileZoneClasses` adds classes per mobile zone.

## Layout

- `lg` and up: one flex line per row, `gap-4`, each `cell` wrapped in a `div` with the column's
  `className`. The column header row is `hidden` below `lg`.
- Below `lg`, plain rows: zones stacked in the fixed order `title`, `meta`, `fill`, `badges`. A
  column without `mobile` is not rendered at all; an empty zone is skipped.
- Selectable rows render `cell` for every column at every width — `mobile` is not used, and the
  cells are not wrapped in the column's `className`, so `cell` has to carry its own width.

## Semantic styles

Every part follows the `Table` `variant`, passed down through context (`table-context.ts`). The
title area, column header and row are cva variants; frame and toolbar are ternaries in
`table.tsx`. `bordered` adds the `Card` frame, `px-4` padding on the header and rows, a
`bg-surface-2` column header and a rounded last row.

Column titles are `text-ink-subtle text-xs font-semibold uppercase`. Rows are separated by
`border-hairline`, none after the last.

States, selectable rows only: hover `bg-surface-2`, checked `bg-primary-soft/40`. Plain rows have
no hover state.
