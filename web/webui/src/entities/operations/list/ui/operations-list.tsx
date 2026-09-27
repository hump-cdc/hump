import { type FC, useCallback } from 'react'

import { useOperationDetailsStore } from '@/entities/operations/details/@x/list'
import { DataTable } from '@/shared/ui/data-table'

import type { Operation } from '../model/types'
import { useOperations } from '../model/use-operations'
import { OperationRetries } from './operation-retries'
import { columns } from './operations-list.columns'

// Panel padding (12 + 16) + border (2) + header (32) + rows (44 each), matching `OperationRetries`.
const detailHeight = (row: Operation) => 62 + (row.retries?.length ?? 0) * 44

export const OperationsList: FC = () => {
    const { rowCount, getRow, onVisibleRangeChange, isLoading } = useOperations()
    const selectedId = useOperationDetailsStore((s) => s.selected?.id ?? null)
    const select = useOperationDetailsStore((s) => s.select)
    const clear = useOperationDetailsStore((s) => s.clear)
    // Stable, so the memoized rows keep their props.
    const onRowSelect = useCallback((_: string, row: Operation) => select(row), [select])

    return (
        <DataTable
            id="operations-table"
            className="h-full"
            columns={columns}
            getRowId={(r) => r.id}
            rowCount={rowCount}
            getRow={getRow}
            onVisibleRangeChange={onVisibleRangeChange}
            isLoading={isLoading}
            isRowExpandable={(r) => !!r.retries?.length}
            getDetail={OperationRetries}
            detailHeight={detailHeight}
            selectedRowId={selectedId}
            onRowSelect={onRowSelect}
            onClearSelection={clear}
        />
    )
}
