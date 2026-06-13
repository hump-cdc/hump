import type { ComponentType, ReactNode } from 'react'

/** A single column definition for {@link DataTable}. */
export type DataTableColumn<T> = {
    /** Row property to read the cell value from (used when `render` is omitted). */
    field?: keyof T & string
    /** Column header label. */
    header?: string
    /**
     * Column width in pixels. Acts as a fixed width in opt-in flex mode (when
     * any column sets `flex`), or as the flex basis when the table auto-fits all
     * columns to its width (the default, when no column sets `flex`).
     */
    width?: number
    /** Minimum width in pixels — the floor a flexing column won't shrink below. */
    minWidth?: number
    /** Maximum width in pixels — caps how far a flexing column can grow. */
    maxWidth?: number
    /**
     * Flex weight: the column fills remaining horizontal space, distributed
     * proportionally across all flexing columns. Overrides `width`. If no column
     * sets `flex`, every column auto-fits the grid width by default.
     */
    flex?: number
    /** Whether the column can be resized by dragging its border. Defaults to `true`. */
    resizable?: boolean
    /** Extra class names applied to the cell. */
    cellClass?: string
    /** Extra class names applied to the header cell. */
    headerClass?: string
    /**
     * Renders the cell body as a React node. Receives the row and returns a
     * `ReactNode`; AG Grid React renders it through its React cellRenderer
     * pipeline, so hooks, context and other components work as expected.
     */
    render?: (row: T) => ReactNode
}

/** A detail component mounted under an expanded row, receiving the parent row. */
export type DetailComponent<T> = ComponentType<{ row: T }>

type DataTableBaseProps<T> = {
    /** Column definitions, left to right. */
    columns: DataTableColumn<T>[]
    /**
     * Stable identifier for this table. When set, user column resizes are
     * persisted to localStorage under this key and restored on mount. Must be
     * unique per table and stable for a given column layout.
     */
    id?: string
    /** Returns a stable, unique id for a row (used for diffing and expansion). */
    getRowId: (row: T) => string
    /** Body row height in pixels. Defaults to `44`. */
    rowHeight?: number
    /** Header row height in pixels. Defaults to `38`. */
    headerHeight?: number
    /**
     * Class names for the wrapper. The grid needs a bounded height for viewport
     * virtualization — set it with a Tailwind height class here (e.g.
     * `h-[560px]`, `h-full`) or via the numeric {@link DataTableBaseProps.height}
     * prop. Defaults to `h-full`.
     */
    className?: string
    /** Explicit wrapper height. Alternative to a height class on `className`. */
    height?: number | string
}

/** In-memory data with optional expandable detail rows. */
export type DataTableClientProps<T> = DataTableBaseProps<T> & {
    /** All rows, held in memory. The grid virtualizes the DOM. */
    data: T[]
    /** Component shown when a row is expanded. Its presence makes rows expandable. */
    getDetail?: DetailComponent<T>
    /** Whether a given row can be expanded. Defaults to `() => !!getDetail`. */
    isRowExpandable?: (row: T) => boolean
    /** Height of the expanded detail row in pixels (or per-row). Defaults to `200`. */
    detailHeight?: number | ((row: T) => number)
    fetchRows?: never
    rowCount?: never
    blockSize?: never
}

/**
 * Lazy, server-driven rows on a windowed client-side model. When `rowCount` is
 * known the grid is seeded with that many placeholder rows, giving a stable
 * scrollbar up front; the blocks overlapping the viewport are fetched on scroll
 * and swapped in (skeleton while loading). Because rows live in the client-side
 * model, they support the same expandable detail rows as client mode.
 */
export type DataTableServerProps<T> = DataTableBaseProps<T> & {
    /** Fetches the rows for a window. `total`, when returned, finalizes the count. */
    fetchRows: (params: { startRow: number; endRow: number }) => Promise<{ rows: T[]; total?: number }>
    /**
     * Total number of rows. When provided, sizes the scrollbar up front and the
     * grid shows placeholders until each row loads. Omit for an unknown total
     * (the list then grows as blocks load).
     */
    rowCount?: number
    /** Rows fetched per request (block/window size). Defaults to `100`. */
    blockSize?: number
    /** Component shown when a row is expanded. Its presence makes rows expandable. */
    getDetail?: DetailComponent<T>
    /** Whether a given row can be expanded. Defaults to `() => !!getDetail`. */
    isRowExpandable?: (row: T) => boolean
    /** Height of the expanded detail row in pixels (or per-row). Defaults to `200`. */
    detailHeight?: number | ((row: T) => number)
    data?: never
}

export type DataTableProps<T> = DataTableClientProps<T> | DataTableServerProps<T>
