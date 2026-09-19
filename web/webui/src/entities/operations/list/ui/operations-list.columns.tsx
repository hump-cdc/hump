import type { FC, PropsWithChildren } from 'react'

import { CircleCheck, CircleX, RotateCcw } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import type { DataTableColumn } from '@/shared/ui/data-table'
import { Sink } from '@/shared/ui/sink'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { formatLatency, formatTime } from '@/shared/utils'

import type { Operation } from '../model/use-operations'

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

const TextFromStatus: FC<
    PropsWithChildren<{
        status: 'delivered' | 'failed' | 'retrying'
        attempts: number
    }>
> = ({ children, status, attempts }) => (
    <span
        className={`font-mono ${status === 'failed' ? 'text-danger' : attempts > 1 ? 'text-amber' : 'text-ink-subtle'}`}
    >
        {children}
    </span>
)

export const columns: DataTableColumn<Operation>[] = [
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
            <Badge className="px-1 uppercase leading-4" variant={opVariant[r.op]}>
                {r.op}
            </Badge>
        ),
    },
    {
        field: 'sink',
        header: 'Sink',
        width: 150,
        cellClass: 'font-mono text-ink-muted',
        render: (r) => <Sink type={r.sink} host={r.sinkHost} />,
    },
    {
        header: 'Status',
        width: 116,
        render: (r) => (
            <div className="flex h-full items-center justify-center">
                <StatusCell row={r} />
            </div>
        ),
    },
    {
        header: 'Att',
        width: 64,
        render: (r) => (
            <TextFromStatus status={r.status} attempts={r.attempts}>
                {r.attempts}×
            </TextFromStatus>
        ),
    },
    {
        field: 'durationMs',
        header: 'Duration',
        width: 84,
        render: (r) => (
            <TextFromStatus status={r.status} attempts={r.attempts}>
                {formatLatency(r.durationMs)}
            </TextFromStatus>
        ),
    },
    {
        header: 'Error',
        flex: 1,
        minWidth: 160,
        cellClass: 'text-danger',
        render: (r) =>
            r.error ? (
                <Tooltip>
                    <TooltipTrigger>
                        <span className="text-danger text-sm">{r.error}</span>
                    </TooltipTrigger>
                    <TooltipContent variant="primary">{r.error}</TooltipContent>
                </Tooltip>
            ) : null,
    },
]
