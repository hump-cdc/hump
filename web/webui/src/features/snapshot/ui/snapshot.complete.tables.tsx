import type { FC } from 'react'

import { CircleCheck } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import type { TableRowColumn } from '@/shared/ui/table'
import { Table, TableRow } from '@/shared/ui/table'
import { formatNumber } from '@/shared/utils'

import { selectedTables } from '../model/selectors'
import type { SnapshotTable, TableProgress } from '../model/types'
import { useSnapshot } from '../model/use-snapshot'

type ResultRowData = { table: SnapshotTable; progress: TableProgress | undefined }

const RESULT_COLUMNS: TableRowColumn<ResultRowData>[] = [
    {
        key: 'status',
        className: 'size-4 shrink-0',
        cell: () => <CircleCheck className="size-4 text-success" />,
        mobile: {
            zone: 'title',
            render: () => <CircleCheck className="size-4 shrink-0 text-success" />,
        },
    },
    {
        key: 'name',
        header: 'Table',
        className: 'flex-1 truncate',
        cell: (d) => <span className="font-mono text-base text-ink">{d.table.name}</span>,
        mobile: {
            zone: 'title',
            render: (d) => <span className="min-w-0 truncate font-mono text-base text-ink">{d.table.name}</span>,
        },
    },
    {
        key: 'rows',
        header: 'Rows',
        className: 'w-30 text-right',
        cell: (d) => (
            <span className="font-mono text-ink-muted text-sm">{formatNumber(d.progress?.rowsRead ?? 0)}</span>
        ),
        mobile: {
            zone: 'meta',
            render: (d) => <span>{formatNumber(d.progress?.rowsRead ?? 0)} rows</span>,
        },
    },
    {
        key: 'items',
        header: 'Items',
        className: 'w-30 text-right',
        cell: (d) => (
            <span className="font-mono text-ink-muted text-sm">{formatNumber(d.progress?.itemsSent ?? 0)}</span>
        ),
        mobile: {
            zone: 'meta',
            render: (d) => <span>{formatNumber(d.progress?.itemsSent ?? 0)} items</span>,
        },
    },
    {
        key: 'sinks',
        header: 'Sinks',
        className: 'w-50 text-right',
        cell: (d) => (
            <div className="flex flex-wrap justify-end gap-1.5">
                {d.table.sinks.map((sink) => (
                    <Badge key={sink} variant="neutral" className="font-mono">
                        {sink}
                    </Badge>
                ))}
            </div>
        ),
        mobile: {
            zone: 'badges',
            render: (d) => (
                <>
                    {d.table.sinks.map((sink) => (
                        <Badge key={sink} variant="neutral" className="font-mono">
                            {sink}
                        </Badge>
                    ))}
                </>
            ),
        },
    },
]

export const CompleteTables: FC = () => {
    const { selectedIds, progress } = useSnapshot()

    const tables = selectedTables(selectedIds)

    return (
        <Table
            columns={RESULT_COLUMNS}
            className="lg:flex-1"
            header={<span className="font-semibold text-ink text-md">Snapshotted tables</span>}
            headerAside={
                <span className="font-medium text-ink-subtle text-sm">{tables.length} tables · all delivered</span>
            }
        >
            {tables.map((table) => (
                <TableRow
                    key={table.id}
                    row={{ table, progress: progress[table.id] }}
                    columns={RESULT_COLUMNS}
                    mobileZoneClasses={{ meta: 'pl-6', badges: 'pl-6' }}
                />
            ))}
        </Table>
    )
}
