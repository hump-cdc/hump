import { DEFAULT_FILL_MIN_WIDTH } from './data-table.consts'
import type { DataTableColumn } from './data-table.types'

/** Width of a fixed column that declares none. */
const DEFAULT_COLUMN_WIDTH = 200

/** User-resized column widths in pixels, keyed by {@link columnId}. */
export type ColumnSizing = Record<string, number>

/** Stable id of the column at an index, which persisted sizing is keyed by. */
export const columnId = (index: number): string => `col_${index}`

/** Whether no column opts into `flex`, so every column auto-fits the table width. */
export const isAutoFill = <T>(columns: DataTableColumn<T>[]): boolean => columns.every((col) => col.flex == null)

/** Whether dragging a column's border changes its width; flex columns always refill instead. */
export const isResizableColumn = <T>(col: DataTableColumn<T>): boolean => col.resizable !== false && col.flex == null

/**
 * Resolves every column's pixel width. Fixed and resized columns keep their
 * width; flexing ones share what is left of `availableWidth` by weight, within
 * their min/max, so the row may overflow (and scroll) but never squash.
 */
export const resolveColumnWidths = <T>(
    columns: DataTableColumn<T>[],
    sizing: ColumnSizing,
    availableWidth: number,
): number[] => {
    const autoFill = isAutoFill(columns)
    const widths = columns.map((col, index) => fixedWidth(col, sizing[columnId(index)], autoFill))
    const flexing = columns.flatMap((col, index) =>
        widths[index] === undefined ? [flexSpec(col, index, autoFill)] : [],
    )
    const fixedTotal = widths.reduce<number>((sum, width) => sum + (width ?? 0), 0)

    for (const [index, width] of distribute(flexing, Math.max(0, availableWidth - fixedTotal))) {
        widths[index] = width
    }

    return widths.map((width) => width ?? 0)
}

/** A flexing column's share parameters. */
type FlexSpec = {
    index: number
    weight: number
    min: number
    max: number
}

/** Width of a column that does not flex, or `undefined` for one that does. */
const fixedWidth = <T>(col: DataTableColumn<T>, resized: number | undefined, autoFill: boolean): number | undefined => {
    if (col.flex != null || (autoFill && resized === undefined)) {
        return undefined
    }

    return clamp(
        resized ?? col.width ?? DEFAULT_COLUMN_WIDTH,
        col.minWidth ?? 0,
        col.maxWidth ?? Number.POSITIVE_INFINITY,
    )
}

/** Weight and bounds of a flexing column; an auto-fitting one never shrinks below its width. */
const flexSpec = <T>(col: DataTableColumn<T>, index: number, autoFill: boolean): FlexSpec => ({
    index,
    weight: col.flex ?? col.width ?? 1,
    min: col.minWidth ?? (autoFill ? (col.width ?? DEFAULT_FILL_MIN_WIDTH) : 0),
    max: col.maxWidth ?? Number.POSITIVE_INFINITY,
})

/**
 * Splits `space` across the flexing columns by weight. A column whose share
 * breaks its bounds is pinned to the bound and the rest is re-split, until every
 * share fits; the rounding remainder goes to the last column still flexing.
 */
const distribute = (specs: FlexSpec[], space: number): Map<number, number> => {
    const result = new Map<number, number>()
    let open = specs
    let remaining = space

    while (open.length > 0) {
        const totalWeight = open.reduce((sum, spec) => sum + spec.weight, 0)
        const snapshot = remaining
        const pinned = open.filter((spec) => {
            const share = (snapshot * spec.weight) / totalWeight
            return share < spec.min || share > spec.max
        })

        if (pinned.length === 0) {
            break
        }

        for (const spec of pinned) {
            const width = Math.round(clamp((snapshot * spec.weight) / totalWeight, spec.min, spec.max))
            result.set(spec.index, width)
            remaining -= width
        }
        open = open.filter((spec) => !pinned.includes(spec))
    }

    const totalWeight = open.reduce((sum, spec) => sum + spec.weight, 0)
    let assigned = 0
    open.forEach((spec, position) => {
        const isLast = position === open.length - 1
        const width = isLast
            ? Math.max(0, Math.round(remaining) - assigned)
            : Math.round((remaining * spec.weight) / totalWeight)
        result.set(spec.index, width)
        assigned += width
    })

    return result
}

/** `value` limited to `[min, max]`. */
const clamp = (value: number, min: number, max: number): number => Math.min(Math.max(value, min), max)
