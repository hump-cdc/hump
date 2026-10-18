# DataTable

A virtualized table with a fixed header, resizable columns, a selectable row and expandable
detail rows, on TanStack Table (columns) and TanStack Virtual (rows). Use it for long or lazily
loaded lists; for a short, fully loaded list pick `@/shared/ui/table`.
Storybook: `Content/Data Table`.

`index.ts` exports the `DataTable` component and the types `DataTableColumn<T>`,
`DataTableProps<T>`, `DataTableRange`, `DetailComponent<T>`. The other `data-table.*` files
(header, row, expand button, skeleton, sizing, column model, column state, consts) are internal.

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

### DataTable

| Prop | Type | Default | Description |
|---|---|---|---|
| `columns` | `DataTableColumn<T>[]` | required | Left to right |
| `getRowId` | `(row: T) => string` | required | Stable unique id; expansion and selection follow it |
| `rowCount` | `number` | required | Total rows; sizes the scrollbar |
| `getRow` | `(index: number) => T \| undefined` | required | `undefined` while the row is not loaded |
| `id` | `string` | — | When set, resized column widths are saved to localStorage and restored on mount |
| `onVisibleRangeChange` | `(range: DataTableRange) => void` | — | Rendered index range, overscan included; `end` is inclusive |
| `isLoading` | `boolean` | `false` | With `rowCount === 0`: 20 skeleton rows instead of `emptyMessage` |
| `getDetail` | `ComponentType<{ row: T }>` | — | Detail component under an expanded row; its presence adds the expand column |
| `isRowExpandable` | `(row: T) => boolean` | every row | Narrows which rows get the chevron |
| `detailHeight` | `number \| ((row: T) => number)` | `200` | Detail panel height, px |
| `selectedRowId` | `string \| null` | — | Highlighted row |
| `onRowSelect` | `(id: string, row: T) => void` | — | Row click; its presence makes rows selectable and the body focusable |
| `onClearSelection` | `() => void` | — | Click in the body outside any row |
| `rowHeight` | `number` | `44` | px, bottom border included |
| `headerHeight` | `number` | `38` | px, bottom border excluded |
| `overscan` | `number` | `10` | Rows rendered beyond each edge |
| `className` | `string` | — | Wrapper classes |
| `height` | `number \| string` | — | Explicit wrapper height, alternative to a height class |
| `emptyMessage` | `ReactNode` | `'No Rows To Show'` | Shown when there are no rows |

### DataTableColumn<T>

| Field | Type | Default | Description |
|---|---|---|---|
| `field` | `keyof T & string` | — | Row property shown when `render` is omitted |
| `header` | `string` | — | Header label |
| `width` | `number` | — | px. Fixed width when any column sets `flex`; flex basis when none does |
| `minWidth` | `number` | — | Floor for flexing and resizing |
| `maxWidth` | `number` | — | Cap for flexing and resizing |
| `flex` | `number` | — | Weight for sharing the remaining width; overrides `width` |
| `resizable` | `boolean` | `true` | Shows the grip on the right border |
| `cellClass` | `string` | — | Extra classes on the cell |
| `headerClass` | `string` | — | Extra classes on the header cell |
| `render` | `(row: T) => ReactNode` | — | Cell body; called only for loaded rows |

A column with neither `render` nor `field` renders an empty cell.

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

No cva variants. Tokens and what they signal:

| Element | Tokens |
|---|---|
| wrapper | `bg-surface-1`, `border-hairline`, `rounded-xl`, `text-ink text-base` |
| header row | `bg-surface-2`, `border-hairline` bottom; titles `text-ink-subtle text-xs font-semibold uppercase tracking-caps` |
| resize grip | short `bg-hairline` divider; `cursor-ew-resize` only where resizing works |
| body row | `bg-surface-1`, `border-hairline` bottom |
| skeleton cell | `bg-surface-3`, `animate-pulse` |
| expand chevron | `text-ink-tertiary`; `text-ink-muted` on hover and when expanded |
| detail panel | `bg-surface-2` gutter around a `bg-canvas`, `border-hairline`, rounded box |
| scrollbar | thin, thumb `--hairline-strong` |

States: row hover `bg-surface-2`; selected row `bg-primary-soft/40` (kept on hover); rows show
`cursor-pointer` only when `onRowSelect` is set. There is no focus ring on the body
(`outline-none`).
