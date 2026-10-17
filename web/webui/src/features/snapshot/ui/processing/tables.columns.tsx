import type { ComponentProps } from 'react'

import { Table2 } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import { NumberFlow } from '@/shared/ui/number-flow'
import { ProgressBar } from '@/shared/ui/progress-bar'
import type { TableRowColumn } from '@/shared/ui/table'

import { snapshotQuery } from '../../model/selectors'
import type { Table as RunTable, TableStatus } from '../../model/types'
import { QueryTooltip } from '../query-tooltip'

type ProcessingRowData = { table: RunTable }

const STATUS_BADGE: Record<TableStatus, ComponentProps<typeof Badge>['variant']> = {
    wait: 'neutral',
    processing: 'active',
    done: 'healthy',
    success: 'healthy',
    error: 'failed',
}

export const PROGRESS_COLUMNS: TableRowColumn<ProcessingRowData>[] = [
    {
        key: 'name',
        header: 'Table',
        className: 'w-42 truncate',
        cell: (d) => (
            <span className="flex min-w-0 items-center gap-2">
                <QueryTooltip query={snapshotQuery(d.table, d.table.where)}>
                    <Table2 className="size-4 text-primary" />
                </QueryTooltip>
                <span className="truncate font-mono text-base text-ink">{d.table.name}</span>
            </span>
        ),
        mobile: {
            zone: 'title',
            render: (d) => (
                <span className="flex min-w-0 flex-1 items-center gap-2">
                    <QueryTooltip query={snapshotQuery(d.table, d.table.where)}>
                        <Table2 className="size-4 text-primary" />
                    </QueryTooltip>
                    <span className="truncate font-mono text-base text-ink">{d.table.name}</span>
                </span>
            ),
        },
    },
    {
        key: 'status',
        header: 'Status',
        className: 'w-28',
        cell: (d) => {
            const { status } = d.table
            return (
                <Badge variant={STATUS_BADGE[status]} dot dotAnimated={status === 'processing'}>
                    {status}
                </Badge>
            )
        },
        mobile: {
            zone: 'title',
            render: (d) => {
                const { status } = d.table
                return (
                    <Badge variant={STATUS_BADGE[status]} dot dotAnimated={status === 'processing'}>
                        {status}
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
            const { status, rowsRead, totalRows } = d.table
            const percent = totalRows === 0 ? 0 : (rowsRead / totalRows) * 100
            return <ProgressBar value={percent} size="sm" variant={status === 'done' ? 'success' : 'primary'} />
        },
        mobile: {
            zone: 'fill',
            render: (d) => {
                const { status, rowsRead, totalRows } = d.table
                const percent = totalRows === 0 ? 0 : (rowsRead / totalRows) * 100
                return <ProgressBar value={percent} size="sm" variant={status === 'done' ? 'success' : 'primary'} />
            },
        },
    },
    {
        key: 'rows',
        header: 'Rows read',
        className: 'w-40 text-right',
        cell: (d) => {
            const { rowsRead } = d.table
            return (
                <span className="font-mono text-ink-muted text-sm">
                    <NumberFlow value={rowsRead} />
                </span>
            )
        },
        mobile: {
            zone: 'meta',
            render: (d) => {
                const { rowsRead } = d.table
                return (
                    <span className="font-mono text-ink-subtle text-sm">
                        <NumberFlow value={rowsRead} /> rows
                    </span>
                )
            },
        },
    },
]
