import type { FC } from 'react'

import { CircleCheck, CircleX, RotateCcw } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import { DataTable, type DataTableColumn } from '@/shared/ui/data-table'
import { formatTime } from '@/shared/utils'

import { fetchOperations, TOTAL_ROWS } from '../model/mock'
import type { Operation } from '../model/types'
import { OperationRetries } from './operation-retries'

const opVariant = {
    insert: 'healthy',
    update: 'active',
    delete: 'failed',
    select: 'lag',
} as const

const statusIcon: Record<Operation['status'], FC> = {
    delivered: () => <CircleCheck className="size-3.75 text-success" />,
    failed: () => <CircleX className="size-3.75 text-danger" />,
    retrying: () => <RotateCcw className="size-3.75 text-amber" />,
}

/** Custom React component used as a cell renderer. */
const StatusCell: FC<{ row: Operation }> = ({ row }) => {
    const Icon = statusIcon[row.status]
    return <Icon />
}

const columns: DataTableColumn<Operation>[] = [
    {
        field: 'time',
        header: 'Time',
        width: 148,
        cellClass: 'font-mono text-sm text-ink-subtle',
        render: (r) => formatTime(r.time),
    },
    { field: 'table', header: 'Table', width: 150, cellClass: 'font-mono text-ink' },
    {
        header: 'Operation',
        width: 104,
        render: (r) => (
            <Badge className="leading-4 px-1" variant={opVariant[r.op]}>
                {r.op}
            </Badge>
        ),
    },
    { field: 'sink', header: 'Sink', width: 150, cellClass: 'font-mono text-ink-muted' },
    {
        header: 'Status',
        width: 116,
        render: (r) => (
            <div className="flex items-center justify-center h-full">
                <StatusCell row={r} />
            </div>
        ),
    },
    {
        header: 'Att',
        width: 64,
        render: (r) => (
            <span className={`font-mono ${r.attempts > 1 ? 'text-amber' : 'text-ink-tertiary'}`}>{r.attempts}×</span>
        ),
    },
    {
        field: 'duration',
        header: 'Dur',
        width: 84,
        render: (r) => (
            <span className={`font-mono ${r.attempts > 1 ? 'text-amber' : 'text-ink-muted'}`}>{r.duration}</span>
        ),
    },
    {
        header: 'Error',
        flex: 1,
        minWidth: 160,
        cellClass: 'text-danger',
        render: (r) => (r.error ? <span className="text-sm text-danger">{r.error}</span> : null),
    },
]

const detailHeight = (row: Operation) => 53 + (row.retries?.length ?? 0) * 41

export const OperationsList: FC = () => (
    <DataTable
        id="operations-table"
        className="h-full"
        columns={columns}
        getRowId={(r) => r.id}
        rowCount={TOTAL_ROWS}
        blockSize={100}
        isRowExpandable={(r) => !!r.retries?.length}
        getDetail={OperationRetries}
        detailHeight={detailHeight}
        fetchRows={fetchOperations}
    />
)
