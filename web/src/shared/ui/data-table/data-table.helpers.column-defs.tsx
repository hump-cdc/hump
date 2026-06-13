import { type ReactNode, useMemo } from 'react'

import type { ColDef } from 'ag-grid-community'
import type { CustomCellRendererProps } from 'ag-grid-react'

import { cn } from '@/shared/utils'

import { DEFAULT_FILL_MIN_WIDTH } from './data-table.consts'
import { ExpandButton } from './data-table.expand-button'
import type { InternalRow } from './data-table.helpers.expandable'
import { Skeleton } from './data-table.skeleton'
import type { DataTableColumn } from './data-table.types'

/** Stable colId for a column at a given index — lets the resize handler target it. */
export const columnId = (index: number) => `col_${index}`

export const defaultColDef: ColDef = {
    resizable: true,
    sortable: false,
    suppressMovable: true,
}

/**
 * Resolves a column's sizing, never emitting `width` and `flex` together:
 * - explicit `flex` → the column fills (proportional weight);
 * - `autoFill` (no column opted into flex) → every column flexes by its width,
 *   so they scale to fill the grid width and scroll (not squash) when narrower;
 * - otherwise → fixed `width`.
 */
const resolveSizing = <T,>(
    col: DataTableColumn<T>,
    autoFill: boolean,
): Pick<ColDef, 'width' | 'flex' | 'minWidth' | 'maxWidth'> => {
    if (col.flex != null) {
        return { flex: col.flex, minWidth: col.minWidth, maxWidth: col.maxWidth }
    }
    if (autoFill) {
        return {
            flex: col.width ?? 1,
            minWidth: col.minWidth ?? col.width ?? DEFAULT_FILL_MIN_WIDTH,
            maxWidth: col.maxWidth,
        }
    }
    return { width: col.width, minWidth: col.minWidth, maxWidth: col.maxWidth }
}

/** Maps public column definitions to AG Grid `ColDef`s over the internal row wrapper. */
export const buildColumnDefs = <T,>(columns: DataTableColumn<T>[], autoFill: boolean): ColDef<InternalRow<T>>[] =>
    columns.map((col, index) => ({
        colId: columnId(index),
        headerName: col.header ?? '',
        ...resolveSizing(col, autoFill),
        resizable: col.resizable ?? true,
        cellClass: col.cellClass,
        headerClass: cn('uppercase tracking-caps', col.headerClass),
        cellRenderer: (params: CustomCellRendererProps<InternalRow<T>>): ReactNode => {
            const data = params.data
            if (!data) {
                return null
            }
            if (data.kind === 'placeholder') {
                return <Skeleton />
            }
            if (data.kind !== 'master') {
                return null
            }
            if (col.render) {
                return col.render(data.row)
            }
            return col.field ? ((data.row[col.field as keyof T] as ReactNode) ?? null) : null
        },
    }))

/** Builds the AG Grid column defs, prepending the expand column when expandable. */
export const useColumnDefs = <T,>(
    columns: DataTableColumn<T>[],
    autoFill: boolean,
    expandable: boolean,
): ColDef<InternalRow<T>>[] =>
    useMemo<ColDef<InternalRow<T>>[]>(() => {
        const mapped = buildColumnDefs(columns, autoFill)
        if (!expandable) {
            return mapped
        }
        const expandCol: ColDef<InternalRow<T>> = {
            colId: '__expand',
            headerName: '',
            width: 36,
            minWidth: 36,
            maxWidth: 36,
            resizable: false,
            sortable: false,
            cellClass: '!p-0',
            cellRenderer: ExpandButton,
        }
        return [expandCol, ...mapped]
    }, [columns, expandable, autoFill])
