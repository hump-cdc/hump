import type { FC } from 'react'

import { DataTable } from '@/shared/ui/data-table'

import { type Operation, useOperations } from '../model/use-operations'
import { OperationRetries } from './operation-retries'
import { columns } from './operations-list.columns'

const detailHeight = (row: Operation) => 53 + (row.retries?.length ?? 0) * 41

export const OperationsList: FC = () => {
    const { fetchRows, rowCount, blockSize, refreshToken } = useOperations()

    return (
        <DataTable
            id="operations-table"
            className="h-full"
            columns={columns}
            getRowId={(r) => r.id}
            fetchRows={fetchRows}
            rowCount={rowCount}
            blockSize={blockSize}
            refreshToken={refreshToken}
            isRowExpandable={(r) => !!r.retries?.length}
            getDetail={OperationRetries}
            detailHeight={detailHeight}
        />
    )
}
