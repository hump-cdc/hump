import type { ComponentProps, FC } from 'react'

import { Badge } from '@/shared/ui/badge'
import { ProgressBar } from '@/shared/ui/progress-bar'
import type { TableRowColumn } from '@/shared/ui/table'
import { Table, TableRow } from '@/shared/ui/table'
import { formatNumber } from '@/shared/utils'

import { selectedTables } from '../model/selectors'
import type { SnapshotTable, TableProgress, TableStatus } from '../model/types'
import { useSnapshot } from '../model/use-snapshot'

type RunningRowData = { table: SnapshotTable; progress: TableProgress | undefined }

const STATUS_LABEL: Record<TableStatus, string> = {
    waiting: 'waiting',
    processing: 'processing',
    done: 'done',
}

const STATUS_BADGE: Record<TableStatus, ComponentProps<typeof Badge>['variant']> = {
    waiting: 'neutral',
    processing: 'active',
    done: 'healthy',
}

const RUNNING_COLUMNS: TableRowColumn<RunningRowData>[] = [
    {
        key: 'name',
        header: 'Table',
        className: 'w-42 truncate',
        cell: (d) => <span className="font-mono text-base text-ink">{d.table.name}</span>,
        mobile: {
            zone: 'title',
            render: (d) => <span className="min-w-0 flex-1 truncate font-mono text-base text-ink">{d.table.name}</span>,
        },
    },
    {
        key: 'status',
        header: 'Status',
        className: 'w-28',
        cell: (d) => {
            const status = d.progress?.status ?? 'waiting'
            return (
                <Badge variant={STATUS_BADGE[status]} dot dotAnimated={status === 'processing'}>
                    {STATUS_LABEL[status]}
                </Badge>
            )
        },
        mobile: {
            zone: 'title',
            render: (d) => {
                const status = d.progress?.status ?? 'waiting'
                return (
                    <Badge variant={STATUS_BADGE[status]} dot dotAnimated={status === 'processing'}>
                        {STATUS_LABEL[status]}
                    </Badge>
                )
            },
        },
    },
    {
        key: 'progress',
        header: 'Progress',
        className: 'flex-1',
        cell: (d) => {
            const status = d.progress?.status ?? 'waiting'
            const rowsRead = d.progress?.rowsRead ?? 0
            const percent = d.table.totalRows === 0 ? 0 : (rowsRead / d.table.totalRows) * 100
            return <ProgressBar value={percent} size="sm" variant={status === 'done' ? 'success' : 'primary'} />
        },
        mobile: {
            zone: 'fill',
            render: (d) => {
                const status = d.progress?.status ?? 'waiting'
                const rowsRead = d.progress?.rowsRead ?? 0
                const percent = d.table.totalRows === 0 ? 0 : (rowsRead / d.table.totalRows) * 100
                return <ProgressBar value={percent} size="sm" variant={status === 'done' ? 'success' : 'primary'} />
            },
        },
    },
    {
        key: 'rows',
        header: 'Rows read / total',
        className: 'w-40 text-right',
        cell: (d) => {
            const rowsRead = d.progress?.rowsRead ?? 0
            return (
                <span className="font-mono text-sm text-ink-muted">
                    {formatNumber(rowsRead)} / {formatNumber(d.table.totalRows)}
                </span>
            )
        },
        mobile: {
            zone: 'meta',
            render: (d) => {
                const rowsRead = d.progress?.rowsRead ?? 0
                return (
                    <span className="font-mono text-sm text-ink-subtle">
                        {formatNumber(rowsRead)} / {formatNumber(d.table.totalRows)} rows
                    </span>
                )
            },
        },
    },
]

export const RunningTables: FC = () => {
    const { selectedIds, progress } = useSnapshot()

    const tables = selectedTables(selectedIds)

    return (
        <Table
            columns={RUNNING_COLUMNS}
            header={
                <div className="flex items-center gap-2.5">
                    <span className="text-md font-semibold text-ink">Tables</span>
                    <Badge variant="neutral">{tables.length}</Badge>
                </div>
            }
            headerAside={
                <div className="flex items-center gap-3.5 text-sm text-ink-subtle">
                    <span className="flex items-center gap-1.5">
                        <span className="size-1.5 rounded-full bg-ink-subtle" />
                        waiting
                    </span>
                    <span className="flex items-center gap-1.5">
                        <span className="size-1.5 rounded-full bg-primary" />
                        processing
                    </span>
                    <span className="flex items-center gap-1.5">
                        <span className="size-1.5 rounded-full bg-success" />
                        done
                    </span>
                </div>
            }
        >
            {tables.map((table) => (
                <TableRow key={table.id} row={{ table, progress: progress[table.id] }} columns={RUNNING_COLUMNS} />
            ))}
        </Table>
    )
}
