import { columnResizingFeature, columnSizingFeature, type Header, tableFeatures } from '@tanstack/react-table'

/** TanStack Table features the column model registers: column widths and the resize drag. */
export const columnFeatures = tableFeatures({ columnSizingFeature, columnResizingFeature })

/** Feature set of the column model, the first type argument of its TanStack types. */
export type ColumnFeatures = typeof columnFeatures

/**
 * Row type of the column model. The model never holds rows (the table renders
 * them by index), so it is an empty record that satisfies TanStack's `RowData`.
 */
export type ColumnModelRow = Record<string, never>

/** One header of the column model, carrying the resize API of its column. */
export type ColumnHeader = Header<ColumnFeatures, ColumnModelRow, unknown>
