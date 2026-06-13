import { useCallback } from 'react'

import type { ColumnResizedEvent, GridReadyEvent } from 'ag-grid-community'

import { type ColumnLocalState, loadColumnState, saveColumnState } from './data-table.column-state'
import type { InternalRow } from './data-table.helpers.expandable'

/** Re-asserts fill columns and persists sizing after the user drags a column border. */
export const useColumnResizedHandler = <T>(flexState: ColumnLocalState[], persistId?: string) =>
    useCallback(
        (event: ColumnResizedEvent<InternalRow<T>>) => {
            if (!event.finished || event.source !== 'uiColumnResized') {
                return
            }
            // Keep the fill column(s) filling after the user drags a border.
            event.api.applyColumnState({ state: flexState })
            // Persist the resulting sizing (flex on fill columns, width on the rest).
            if (persistId) {
                const state: ColumnLocalState[] = event.api
                    .getColumnState()
                    .filter(
                        (col): col is ColumnLocalState & { colId: string } => !!col.colId && col.colId !== '__expand',
                    )
                    .map((col) => ({ colId: col.colId, width: col.width, flex: col.flex }))
                saveColumnState(persistId, state)
            }
        },
        [flexState, persistId],
    )

/** Restores persisted column sizing once the grid is ready. */
export const useGridReadyHandler = <T>(columnCount: number, persistId?: string) =>
    useCallback(
        (event: GridReadyEvent<InternalRow<T>>) => {
            if (!persistId) {
                return
            }
            const saved = loadColumnState(persistId)
            // Guard against a changed column layout: positional colIds only line up
            // when the saved entry count matches the current data columns.
            if (saved && saved.length === columnCount) {
                event.api.applyColumnState({ state: saved, applyOrder: false })
            }
        },
        [persistId, columnCount],
    )
