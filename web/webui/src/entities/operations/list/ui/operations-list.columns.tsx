import type { FC, PropsWithChildren } from 'react'

import type { DataTableColumn } from '@/shared/ui/data-table'
import { OperationBadge } from '@/shared/ui/operation-badge'
import { OperationStatusBadge } from '@/shared/ui/operation-status-badge'
import { Sink } from '@/shared/ui/sink'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { formatLatency, formatTime } from '@/shared/utils'

import type { Operation } from '../model/types'

const TextFromStatus: FC<
    PropsWithChildren<{
        status: Operation['status']
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
        render: (r) => <OperationBadge op={r.op} />,
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
                <OperationStatusBadge variant="compact" status={r.status} attempts={r.attempts} />
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
                    {/* A plain span, not Radix's default button, so the error is no Tab stop. */}
                    <TooltipTrigger asChild>
                        <span className="text-danger text-sm">{r.error}</span>
                    </TooltipTrigger>
                    <TooltipContent variant="primary">{r.error}</TooltipContent>
                </Tooltip>
            ) : null,
    },
]
