import type { ComponentType, ReactNode } from 'react'

/** A single column definition for {@link DataTable}. */
export type DataTableColumn<T> = {
    /** Row property to read the cell value from (used when `render` is omitted). */
    field?: keyof T & string
    /** Column header label. */
    header?: string
    /**
     * Column width in pixels. Acts as a fixed width when any column sets `flex`,
     * or as the flex basis when the table auto-fits all columns to its width (the
     * default, when no column sets `flex`).
     */
    width?: number
    /** Minimum width in pixels — the floor a flexing or resized column won't shrink below. */
    minWidth?: number
    /** Maximum width in pixels — caps how far a flexing or resized column can grow. */
    maxWidth?: number
    /**
     * Flex weight: the column fills remaining horizontal space, distributed
     * proportionally across all flexing columns. Overrides `width`.
     */
    flex?: number
    /** Whether the column can be resized by dragging its border. Defaults to `true`. */
    resizable?: boolean
    /** Extra class names applied to the cell. */
    cellClass?: string
    /** Extra class names applied to the header cell. */
    headerClass?: string
    /** Renders the cell body; the row is always loaded when this is called. */
    render?: (row: T) => ReactNode
}

/** A detail component mounted under an expanded row, receiving the parent row. */
export type DetailComponent<T> = ComponentType<{ row: T }>

/** Index range of the rendered rows, overscan included; `end` is inclusive. */
export type DataTableRange = {
    start: number
    end: number
}

/**
 * Props of the virtualized table. Rows are sparse: the table asks `getRow` for
 * each rendered index and draws a skeleton for any it gets `undefined` for, so
 * the owner can load them on demand from `onVisibleRangeChange`.
 */
export type DataTableProps<T> = {
    /** Column definitions, left to right. */
    columns: DataTableColumn<T>[]
    /**
     * Stable identifier for this table. When set, user column resizes are
     * persisted to localStorage under this key and restored on mount.
     */
    id?: string
    /** Returns a stable, unique id for a row (used to keep it expanded as it moves). */
    getRowId: (row: T) => string
    /** Total number of rows; sizes the scrollbar up front. */
    rowCount: number
    /** Row at an index, or `undefined` while it is not loaded yet. */
    getRow: (index: number) => T | undefined
    /** Called whenever the rendered index range changes. */
    onVisibleRangeChange?: (range: DataTableRange) => void
    /** Shows skeleton rows instead of the empty message while `rowCount` is not known yet. */
    isLoading?: boolean
    /** Component shown when a row is expanded. Its presence makes rows expandable. */
    getDetail?: DetailComponent<T>
    /** Whether a given row can be expanded. Defaults to `() => !!getDetail`. */
    isRowExpandable?: (row: T) => boolean
    /** Id of the highlighted row, or `null`/`undefined` when none is selected. */
    selectedRowId?: string | null
    /** Called with a row's id and the row when the row is clicked. Its presence makes rows selectable. */
    onRowSelect?: (id: string, row: T) => void
    /** Called when the body is clicked outside any row, e.g. the empty space below the last one. */
    onClearSelection?: () => void
    /** Height of the expanded detail panel in pixels (or per-row). Defaults to `200`. */
    detailHeight?: number | ((row: T) => number)
    /** Body row height in pixels. Defaults to `44`. */
    rowHeight?: number
    /** Header row height in pixels. Defaults to `38`. */
    headerHeight?: number
    /** Rows rendered beyond each edge of the viewport. Defaults to `10`. */
    overscan?: number
    /**
     * Class names for the wrapper. The table needs a bounded height for
     * virtualization — set it here (e.g. `h-full`) or via `height`.
     */
    className?: string
    /** Explicit wrapper height. Alternative to a height class on `className`. */
    height?: number | string
    /** Shown in the body when there are no rows. */
    emptyMessage?: ReactNode
}
