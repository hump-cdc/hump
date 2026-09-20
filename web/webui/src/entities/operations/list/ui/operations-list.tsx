import type { FC } from 'react'

import { DataTable } from '@/shared/ui/data-table'

import type { Operation } from '../model/types'
import { useOperations } from '../model/use-operations'
import { OperationRetries } from './operation-retries'
import { columns } from './operations-list.columns'

const detailHeight = (row: Operation) => 53 + (row.retries?.length ?? 0) * 41

export const OperationsList: FC = () => {
    const { rowCount, getRow, onVisibleRangeChange, isLoading } = useOperations()

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
        />
    )
}
