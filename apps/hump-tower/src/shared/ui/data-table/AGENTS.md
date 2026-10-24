# DataTable

A virtualized table with a fixed header, resizable columns, a selectable row and expandable
detail rows, on TanStack Table (columns) and TanStack Virtual (rows). Use it for long or lazily
loaded lists; for a short, fully loaded list pick `@/shared/ui/table`.
Storybook: `Content/Data Table`.

`index.ts` exports `DataTable` and the types of `data-table.types.ts`; the other `data-table.*`
files are internal.

## Usage

```tsx
import { DataTable, type DataTableColumn, type DataTableRange } from '@/shared/ui/data-table'
import { formatTime } from '@/shared/utils'

const columns: DataTableColumn<Operation>[] = [
    {
        field: 'time',
        header: 'Time',
        width: 148,
        cellClass: 'font-mono text-sm text-ink-subtle',
        render: (r) => formatTime(r.time), // time is a Date; without render the raw field is the cell
    },
    { field: 'sink', header: 'Sink', flex: 1, minWidth: 120 },
    { header: 'Error', flex: 2, minWidth: 160, render: (r) => r.error ?? null },
]

<DataTable
    id="operations-table"
    className="h-full"
    columns={columns}
    getRowId={(r) => r.id}
    rowCount={rowCount}
    getRow={getRow}
    onVisibleRangeChange={onVisibleRangeChange}
    isLoading={isLoading}
    getDetail={OperationRetries}
    isRowExpandable={(r) => !!r.retries?.length}
    detailHeight={detailHeight}
    selectedRowId={selectedId}
    onRowSelect={(id) => select(id)}
    onClearSelection={() => select(null)}
/>
```

The table never fetches. It renders `rowCount` rows by index, asks `getRow(index)` for each
rendered one and draws skeleton cells where that returns `undefined`; the owner loads the missing
rows from `onVisibleRangeChange`.

## Props

The props and column fields are documented in `data-table.types.ts`. What they do together:

- `getRowId` must be stable and unique: expansion and selection follow it.
- `id` turns on width persistence (see Resizing).
- `onVisibleRangeChange` gets the rendered range, overscan included; `end` is inclusive.
- `isLoading` with `rowCount === 0` shows 20 skeleton rows instead of `emptyMessage`.
- `getDetail` adds the expand column; `isRowExpandable` narrows which rows get the chevron.
- `onRowSelect` makes rows selectable and the body focusable; `onClearSelection` fires on a
  click in the body outside any row.
- `rowHeight` includes the row's bottom border, `headerHeight` excludes it.
- A column with neither `render` nor `field` renders an empty cell; `render` is called only for
  loaded rows.

## Constraints

- **Bounded height.** The wrapper is `h-full`; give it a sized parent, a height class or `height`.
  The scroller is absolutely positioned, so rows never size the table.
- **Exact sizes.** Rows are not measured: a row is `rowHeight`, plus `detailHeight` when expanded.
  Cells are single-line (`whitespace-nowrap`, ellipsis); a detail taller than `detailHeight` is
  clipped.
- **Controlled selection, uncontrolled expansion.** Selection is `selectedRowId` + `onRowSelect`;
  the set of expanded rows is internal state keyed by row id, with no prop to read or set it.
- **Stable callbacks.** Keep `getRow` and `onVisibleRangeChange` referentially stable
  (`useCallback`): a new `getRow` re-measures the virtualizer, a new `onVisibleRangeChange` is
  called again with the current range.
- **Column widths.** Without any `flex` column all columns auto-fit the table width, weighted by
  `width`, never below `minWidth ?? width ?? 80`. With a `flex` column the others are fixed
  (`width`, default `200`). When the minimums do not fit, the body scrolls horizontally and the
  header follows.
- **Resizing.** Only non-flex columns with `resizable !== false` react to the drag; a flex column
  shows the grip but refills. Widths persist under `hump:data-table:<id>:columns`, keyed by
  column position (`col_<index>`) — reordering or inserting columns re-maps saved widths.
- **Keyboard.** With `onRowSelect` the body is a single Tab stop: Up/Down select the previous/next
  row and scroll it into view, Space toggles the detail of the selected row. The chevron is
  `tabIndex={-1}`; clicking it expands without selecting. Keys typed inside a detail panel are
  left to it.

## Semantic styles

No cva variants.

- Selected row: `bg-primary-soft/40`, kept on hover (other rows hover `bg-surface-2`).
- Rows show `cursor-pointer` only when `onRowSelect` is set; the resize grip shows
  `cursor-ew-resize` only where resizing works.
- The detail panel is a `bg-canvas` box in a `bg-surface-2` gutter.
- There is no focus ring on the body (`outline-none`).
