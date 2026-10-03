import type { FC } from 'react'

import { CircleCheck, CircleX } from 'lucide-react'

import { Sink } from '@/shared/ui/sink'
import type { TableRowColumn } from '@/shared/ui/table'
import { formatNumber, formatNumberCompact } from '@/shared/utils'

import type { Sink as RunSink, Table as RunTable } from '../../model/types'

type ResultRowData = { table: RunTable }

const StatusIcon: FC<{ table: RunTable }> = ({ table }) =>
    table.status === 'error' ? (
        <CircleX className="size-4 shrink-0 text-danger" aria-label={table.error} />
    ) : (
        <CircleCheck className="size-4 shrink-0 text-success" />
    )

const SinkCount: FC<{ sink: RunSink }> = ({ sink }) => (
    <div className="flex items-center gap-2">
        <Sink type={sink.type} host={sink.host} className="min-w-0" />
        <span className="rounded-full border border-hairline bg-surface-2 px-2 py-0.5 font-medium font-mono text-ink text-xs tabular-nums">
            {formatNumberCompact(sink.count)}
        </span>
    </div>
)

export const RESULT_COLUMNS: TableRowColumn<ResultRowData>[] = [
    {
        key: 'status',
        className: 'size-4 shrink-0',
        cell: (d) => <StatusIcon table={d.table} />,
        mobile: {
            zone: 'title',
            render: (d) => <StatusIcon table={d.table} />,
        },
    },
    {
        key: 'name',
        header: 'Table',
        className: 'flex-1 truncate',
        cell: (d) => (
            <div className="flex min-w-0 flex-col gap-0.5">
                <span className="truncate font-mono text-base text-ink">{d.table.name}</span>
                {d.table.error && <span className="truncate text-danger text-sm">{d.table.error}</span>}
            </div>
        ),
        mobile: {
            zone: 'title',
            render: (d) => <span className="min-w-0 truncate font-mono text-base text-ink">{d.table.name}</span>,
        },
    },
    {
        key: 'rows',
        header: 'Rows',
        className: 'w-30 text-right',
        cell: (d) => <span className="font-mono text-ink-muted text-sm">{formatNumber(d.table.rowsRead)}</span>,
        mobile: {
            zone: 'meta',
            render: (d) => <span>{formatNumber(d.table.rowsRead)} rows</span>,
        },
    },
    {
        key: 'events',
        header: 'Events',
        className: 'w-30 text-right',
        cell: (d) => (
            <span className="font-mono text-ink-muted text-sm">
                {formatNumber(d.table.sinks.reduce((sum, sink) => sum + sink.count, 0))}
            </span>
        ),
        mobile: {
            zone: 'meta',
            render: (d) => <span>{formatNumber(d.table.sinks.reduce((sum, sink) => sum + sink.count, 0))} items</span>,
        },
    },
    {
        key: 'sinks',
        header: 'Sinks',
        className: 'w-60 text-right',
        cell: (d) => (
            <div className="flex flex-col items-end gap-1.5">
                {d.table.sinks.map((sink) => (
                    <SinkCount key={`${sink.type} ${sink.host} ${sink.target}`} sink={sink} />
                ))}
            </div>
        ),
        mobile: {
            zone: 'badges',
            render: (d) => (
                <>
                    {d.table.sinks.map((sink) => (
                        <Sink key={`${sink.type} ${sink.host} ${sink.target}`} type={sink.type} host={sink.host} />
                    ))}
                </>
            ),
        },
    },
]
