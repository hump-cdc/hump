import type { ReactElement } from 'react'

import {
    CellStyleModule,
    ClientSideRowModelModule,
    ColumnApiModule,
    type GetRowIdParams,
    ModuleRegistry,
    RowApiModule,
    ValidationModule,
    type ViewportChangedEvent,
} from 'ag-grid-community'
import { AgGridReact } from 'ag-grid-react'

import { cn } from '@/shared/utils'

import { DEFAULT_DETAIL_HEIGHT, DEFAULT_HEADER_HEIGHT, DEFAULT_ROW_HEIGHT } from './data-table.consts'
import { defaultColDef, useColumnDefs } from './data-table.helpers.column-defs'
import type { InternalRow } from './data-table.helpers.expandable'
import { useColumnResizedHandler, useGridReadyHandler } from './data-table.helpers.persistence'
import { useExpandContext, useFullWidthCellRenderer, useRowHeight } from './data-table.helpers.rows'
import { useAutoFill, useFlexState } from './data-table.helpers.sizing'
import { dataTableTheme } from './data-table.theme'
import type { DataTableColumn, DetailComponent } from './data-table.types'

ModuleRegistry.registerModules([
    ClientSideRowModelModule,
    CellStyleModule,
    ColumnApiModule,
    RowApiModule,
    ...(process.env.NODE_ENV !== 'production' ? [ValidationModule] : []),
])

export type GridViewProps<T> = {
    columns: DataTableColumn<T>[]
    rows: InternalRow<T>[]
    toggle: (id: string) => void
    getDetail?: DetailComponent<T>
    detailHeight?: number | ((row: T) => number)
    rowHeight?: number
    headerHeight?: number
    className?: string
    height?: number | string
    /** localStorage key for persisting column resize; omit to disable persistence. */
    persistId?: string
    onViewportChanged?: (event: ViewportChangedEvent<InternalRow<T>>) => void
}

/** Renders the AG Grid given a pre-flattened list of internal rows. */
export const GridView = <T,>({
    columns,
    rows,
    toggle,
    getDetail,
    detailHeight = DEFAULT_DETAIL_HEIGHT,
    rowHeight = DEFAULT_ROW_HEIGHT,
    headerHeight = DEFAULT_HEADER_HEIGHT,
    className,
    height,
    persistId,
    onViewportChanged,
}: GridViewProps<T>): ReactElement => {
    const expandable = !!getDetail
    const context = useExpandContext(toggle)
    const autoFill = useAutoFill(columns)
    const columnDefs = useColumnDefs(columns, autoFill, expandable)
    const flexState = useFlexState(columns)
    const handleColumnResized = useColumnResizedHandler<T>(flexState, persistId)
    const handleGridReady = useGridReadyHandler<T>(columns.length, persistId)
    const fullWidthCellRenderer = useFullWidthCellRenderer(getDetail)
    const getRowHeight = useRowHeight(detailHeight, rowHeight)

    return (
        <div
            className={cn(
                'h-full w-full',
                '**:[[class*=scroll-viewport]]:scrollbar-thin **:[[class*=scroll-viewport]]:[scrollbar-color:var(--hairline-strong)_transparent]',
                className,
            )}
            style={height != null ? { height } : undefined}
        >
            <AgGridReact<InternalRow<T>>
                theme={dataTableTheme}
                rowData={rows}
                columnDefs={columnDefs}
                defaultColDef={defaultColDef}
                getRowId={(params: GetRowIdParams<InternalRow<T>>) => params.data.id}
                context={context}
                isFullWidthRow={(params) => params.rowNode.data?.kind === 'detail'}
                fullWidthCellRenderer={fullWidthCellRenderer}
                getRowHeight={getRowHeight}
                rowHeight={rowHeight}
                headerHeight={headerHeight}
                onViewportChanged={onViewportChanged}
                onGridReady={persistId ? handleGridReady : undefined}
                onColumnResized={flexState.length > 0 || persistId ? handleColumnResized : undefined}
                suppressCellFocus
            />
        </div>
    )
}
