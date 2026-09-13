import type { FC } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { CircleCheck, CircleX, RotateCcw } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'

import { DataTable } from './data-table'
import type { DataTableColumn } from './data-table.types'

type Attempt = {
    attempt: number
    started: string
    duration: string
    result: 'failed' | 'delivered'
    error?: string
}

type Operation = {
    id: string
    time: string
    table: string
    op: 'insert' | 'update' | 'delete' | 'select'
    sink: string
    status: 'delivered' | 'failed' | 'retrying'
    attempts: number
    duration: string
    error?: string
    retries?: Attempt[]
}

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
    { field: 'time', header: 'Time', width: 148, cellClass: 'font-mono text-sm text-ink-subtle' },
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
    { field: 'sink', header: 'Sink', flex: 1, minWidth: 120, cellClass: 'font-mono text-ink-muted' },
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
        flex: 2,
        minWidth: 160,
        render: (r) => (r.error ? <span className="text-sm text-danger">{r.error}</span> : null),
    },
]

/** Nested detail table shown when a row is expanded. */
const OperationRetries: FC<{ row: Operation }> = ({ row }) => (
    <table className="w-full border-collapse">
        <thead>
            <tr className="border-b border-hairline bg-surface-1 text-left text-2xs font-semibold tracking-caps text-ink-subtle uppercase">
                {['Attempt', 'Started', 'Duration', 'Result', 'Error'].map((h) => (
                    <th key={h} className="px-3.5 py-2 font-semibold">
                        {h}
                    </th>
                ))}
            </tr>
        </thead>
        <tbody>
            {row.retries?.map((a, index) => (
                <tr
                    key={a.attempt}
                    className={index < (row.retries?.length ?? 0) - 1 ? 'border-b border-hairline' : undefined}
                >
                    <td className="px-3.5 py-2.5 font-mono text-ink-muted">#{a.attempt}</td>
                    <td className="px-3.5 py-2.5 font-mono text-sm text-ink-subtle">{a.started}</td>
                    <td className="px-3.5 py-2.5 font-mono text-ink-muted">{a.duration}</td>
                    <td className="px-3.5 py-2.5">
                        <Badge variant={a.result === 'failed' ? 'failed' : 'healthy'}>{a.result}</Badge>
                    </td>
                    <td className="px-3.5 py-2.5 text-sm text-danger">{a.error}</td>
                </tr>
            ))}
        </tbody>
    </table>
)

const operations: Operation[] = [
    {
        id: '1',
        time: '10:42:01.508',
        table: 'public.users',
        op: 'update',
        sink: 'rabbitmq1',
        status: 'retrying',
        attempts: 3,
        duration: '7ms',
        retries: [
            { attempt: 1, started: '10:42:01.318', duration: '18ms', result: 'failed', error: 'connection reset' },
            { attempt: 2, started: '10:42:01.392', duration: '22ms', result: 'failed', error: 'connection reset' },
            { attempt: 3, started: '10:42:01.508', duration: '7ms', result: 'delivered' },
        ],
    },
    {
        id: '2',
        time: '10:42:01.290',
        table: 'public.posts',
        op: 'insert',
        sink: 'kafka1',
        status: 'delivered',
        attempts: 1,
        duration: '5ms',
    },
    {
        id: '3',
        time: '10:42:01.271',
        table: 'public.events',
        op: 'delete',
        sink: 'http1',
        status: 'delivered',
        attempts: 2,
        duration: '22ms',
        retries: [
            { attempt: 1, started: '10:42:01.249', duration: '14ms', result: 'failed', error: 'timeout' },
            { attempt: 2, started: '10:42:01.271', duration: '22ms', result: 'delivered' },
        ],
    },
    {
        id: '4',
        time: '10:42:01.238',
        table: 'public.orders',
        op: 'insert',
        sink: 'kafka1',
        status: 'delivered',
        attempts: 1,
        duration: '6ms',
    },
    {
        id: '5',
        time: '10:42:01.204',
        table: 'public.sessions',
        op: 'update',
        sink: 'http1',
        status: 'failed',
        attempts: 5,
        duration: '120ms',
        error: '503 service unavailable · retry scheduled',
        retries: [
            {
                attempt: 1,
                started: '10:42:01.100',
                duration: '30ms',
                result: 'failed',
                error: '503 service unavailable',
            },
            {
                attempt: 2,
                started: '10:42:01.140',
                duration: '28ms',
                result: 'failed',
                error: '503 service unavailable',
            },
            {
                attempt: 3,
                started: '10:42:01.180',
                duration: '32ms',
                result: 'failed',
                error: '503 service unavailable',
            },
        ],
    },
    {
        id: '6',
        time: '10:42:01.180',
        table: 'public.comments',
        op: 'insert',
        sink: 'nats-core',
        status: 'delivered',
        attempts: 1,
        duration: '4ms',
    },
    {
        id: '7',
        time: '10:42:01.142',
        table: 'public.events',
        op: 'insert',
        sink: 'rmq-stream',
        status: 'delivered',
        attempts: 1,
        duration: '9ms',
    },
    {
        id: '8',
        time: '10:42:01.118',
        table: 'public.users',
        op: 'update',
        sink: 'nats-core',
        status: 'delivered',
        attempts: 1,
        duration: '3ms',
    },
    {
        id: '9',
        time: '10:42:01.094',
        table: 'public.posts',
        op: 'update',
        sink: 'rabbitmq1',
        status: 'delivered',
        attempts: 2,
        duration: '11ms',
        retries: [
            { attempt: 1, started: '10:42:01.071', duration: '15ms', result: 'failed', error: 'connection reset' },
            { attempt: 2, started: '10:42:01.094', duration: '11ms', result: 'delivered' },
        ],
    },
    {
        id: '10',
        time: '10:42:01.061',
        table: 'public.events',
        op: 'insert',
        sink: 'http1',
        status: 'delivered',
        attempts: 1,
        duration: '18ms',
    },
    {
        id: '11',
        time: '10:42:01.040',
        table: 'public.orders',
        op: 'delete',
        sink: 'kafka1',
        status: 'delivered',
        attempts: 1,
        duration: '6ms',
    },
    {
        id: '12',
        time: '10:42:00.998',
        table: 'public.sessions',
        op: 'insert',
        sink: 'rmq-stream',
        status: 'delivered',
        attempts: 1,
        duration: '5ms',
    },
    {
        id: '13',
        time: '10:42:00.972',
        table: 'public.comments',
        op: 'update',
        sink: 'http1',
        status: 'failed',
        attempts: 4,
        duration: '88ms',
        error: 'timeout after 30s',
        retries: [
            { attempt: 1, started: '10:42:00.880', duration: '30ms', result: 'failed', error: 'timeout after 30s' },
            { attempt: 2, started: '10:42:00.920', duration: '30ms', result: 'failed', error: 'timeout after 30s' },
        ],
    },
    {
        id: '14',
        time: '10:42:00.951',
        table: 'public.users',
        op: 'insert',
        sink: 'kafka1',
        status: 'delivered',
        attempts: 1,
        duration: '7ms',
    },
    {
        id: '15',
        time: '10:42:00.917',
        table: 'public.payments',
        op: 'insert',
        sink: 'nats-core',
        status: 'delivered',
        attempts: 1,
        duration: '4ms',
    },
]

// 3× the base set; copies get unique ids but keep their retries (still expandable).
const expandableRows: Operation[] = Array.from({ length: 3 }, (_, copy) =>
    operations.map((op) => (copy === 0 ? op : { ...op, id: `${op.id}-${copy}` })),
).flat()

const detailHeight = (row: Operation) => 53 + (row.retries?.length ?? 0) * 41

const meta: Meta = {
    title: 'UI Kit/Data Table',
    parameters: { disableSideBySide: true, disablePadding: true },
}

export default meta

/** In-memory data with expandable retry-attempt detail rows. */
export const Client: StoryObj = {
    name: 'Client',
    render: () => (
        <div className="h-screen w-full">
            <DataTable
                id="operations-expandable"
                className="h-full"
                columns={columns}
                data={expandableRows}
                getRowId={(r) => r.id}
                isRowExpandable={(r) => !!r.retries?.length}
                getDetail={OperationRetries}
                detailHeight={detailHeight}
            />
        </div>
    ),
}

const TOTAL_ROWS = 10_000

/**
 * Lazy, server-driven rows: `rowCount` sizes the scrollbar to 10k up front and
 * the grid shows placeholders until each block loads on scroll. Each row carries
 * its own `retries`, so rows with retries are expandable just like client mode.
 */
export const ServerSide: StoryObj = {
    name: 'Server',
    render: () => (
        <div className="h-screen w-full">
            <DataTable
                id="operations-server"
                className="h-full"
                columns={columns}
                getRowId={(r) => r.id}
                rowCount={TOTAL_ROWS}
                blockSize={100}
                isRowExpandable={(r) => !!r.retries?.length}
                getDetail={OperationRetries}
                detailHeight={detailHeight}
                fetchRows={({ startRow, endRow }) =>
                    new Promise((resolve) => {
                        setTimeout(() => {
                            const count = Math.max(0, Math.min(endRow, TOTAL_ROWS) - startRow)
                            const rows: Operation[] = Array.from({ length: count }, (_, i) => {
                                const index = startRow + i
                                const template = operations[index % operations.length]
                                // Keep `retries` from the template so some rows are expandable.
                                return { ...template, id: String(index), time: `#${index}` }
                            })
                            resolve({ rows, total: TOTAL_ROWS })
                        }, 350)
                    })
                }
            />
        </div>
    ),
}
