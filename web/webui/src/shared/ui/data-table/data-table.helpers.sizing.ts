import { useMemo } from 'react'

import type { ColumnLocalState } from './data-table.column-state'
import { columnId } from './data-table.helpers.column-defs'
import type { DataTableColumn } from './data-table.types'

/** True when no column declares an explicit flex, so columns should auto-fill. */
export const useAutoFill = <T>(columns: DataTableColumn<T>[]): boolean =>
    useMemo(() => columns.every((col) => col.flex == null), [columns])

/** Designated fill columns (those with an explicit flex), re-asserted after a resize. */
export const useFlexState = <T>(columns: DataTableColumn<T>[]): ColumnLocalState[] =>
    useMemo<ColumnLocalState[]>(
        () => columns.flatMap((col, index) => (col.flex != null ? [{ colId: columnId(index), flex: col.flex }] : [])),
        [columns],
    )
