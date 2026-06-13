import type { Operation } from './types'

/** Parses an `HH:mm:ss.SSS` time-of-day into a `Date` on a fixed mock day (UTC). */
const t = (hms: string): Date => new Date(`2026-06-06T${hms}Z`)

const operations: Operation[] = [
    {
        id: '1',
        time: t('10:42:01.508'),
        table: 'public.users',
        op: 'update',
        sink: 'rabbitmq1',
        status: 'retrying',
        attempts: 3,
        duration: '7ms',
        retries: [
            { attempt: 1, started: t('10:42:01.318'), duration: '18ms', result: 'failed', error: 'connection reset' },
            { attempt: 2, started: t('10:42:01.392'), duration: '22ms', result: 'failed', error: 'connection reset' },
            { attempt: 3, started: t('10:42:01.508'), duration: '7ms', result: 'delivered' },
        ],
    },
    {
        id: '2',
        time: t('10:42:01.290'),
        table: 'public.posts',
        op: 'insert',
        sink: 'kafka1',
        status: 'delivered',
        attempts: 1,
        duration: '5ms',
    },
    {
        id: '3',
        time: t('10:42:01.271'),
        table: 'public.events',
        op: 'delete',
        sink: 'http1',
        status: 'delivered',
        attempts: 2,
        duration: '22ms',
        retries: [
            { attempt: 1, started: t('10:42:01.249'), duration: '14ms', result: 'failed', error: 'timeout' },
            { attempt: 2, started: t('10:42:01.271'), duration: '22ms', result: 'delivered' },
        ],
    },
    {
        id: '4',
        time: t('10:42:01.238'),
        table: 'public.orders',
        op: 'insert',
        sink: 'kafka1',
        status: 'delivered',
        attempts: 1,
        duration: '6ms',
    },
    {
        id: '5',
        time: t('10:42:01.204'),
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
                started: t('10:42:01.100'),
                duration: '30ms',
                result: 'failed',
                error: '503 service unavailable',
            },
            {
                attempt: 2,
                started: t('10:42:01.140'),
                duration: '28ms',
                result: 'failed',
                error: '503 service unavailable',
            },
            {
                attempt: 3,
                started: t('10:42:01.180'),
                duration: '32ms',
                result: 'failed',
                error: '503 service unavailable',
            },
        ],
    },
    {
        id: '6',
        time: t('10:42:01.180'),
        table: 'public.comments',
        op: 'insert',
        sink: 'nats-core',
        status: 'delivered',
        attempts: 1,
        duration: '4ms',
    },
    {
        id: '7',
        time: t('10:42:01.142'),
        table: 'public.events',
        op: 'insert',
        sink: 'rmq-stream',
        status: 'delivered',
        attempts: 1,
        duration: '9ms',
    },
    {
        id: '8',
        time: t('10:42:01.118'),
        table: 'public.users',
        op: 'update',
        sink: 'nats-core',
        status: 'delivered',
        attempts: 1,
        duration: '3ms',
    },
    {
        id: '9',
        time: t('10:42:01.094'),
        table: 'public.posts',
        op: 'update',
        sink: 'rabbitmq1',
        status: 'delivered',
        attempts: 2,
        duration: '11ms',
        retries: [
            { attempt: 1, started: t('10:42:01.071'), duration: '15ms', result: 'failed', error: 'connection reset' },
            { attempt: 2, started: t('10:42:01.094'), duration: '11ms', result: 'delivered' },
        ],
    },
    {
        id: '10',
        time: t('10:42:01.061'),
        table: 'public.events',
        op: 'insert',
        sink: 'http1',
        status: 'delivered',
        attempts: 1,
        duration: '18ms',
    },
    {
        id: '11',
        time: t('10:42:01.040'),
        table: 'public.orders',
        op: 'delete',
        sink: 'kafka1',
        status: 'delivered',
        attempts: 1,
        duration: '6ms',
    },
    {
        id: '12',
        time: t('10:42:00.998'),
        table: 'public.sessions',
        op: 'insert',
        sink: 'rmq-stream',
        status: 'delivered',
        attempts: 1,
        duration: '5ms',
    },
    {
        id: '13',
        time: t('10:42:00.972'),
        table: 'public.comments',
        op: 'update',
        sink: 'http1',
        status: 'failed',
        attempts: 4,
        duration: '88ms',
        error: 'timeout after 30s',
        retries: [
            { attempt: 1, started: t('10:42:00.880'), duration: '30ms', result: 'failed', error: 'timeout after 30s' },
            { attempt: 2, started: t('10:42:00.920'), duration: '30ms', result: 'failed', error: 'timeout after 30s' },
        ],
    },
    {
        id: '14',
        time: t('10:42:00.951'),
        table: 'public.users',
        op: 'insert',
        sink: 'kafka1',
        status: 'delivered',
        attempts: 1,
        duration: '7ms',
    },
    {
        id: '15',
        time: t('10:42:00.917'),
        table: 'public.payments',
        op: 'insert',
        sink: 'nats-core',
        status: 'delivered',
        attempts: 1,
        duration: '4ms',
    },
]

export const TOTAL_ROWS = 10_000

/** Newest row's timestamp; generated rows step back from here. */
const BASE_TIME = t('10:42:01.508').getTime()
/** Spacing between consecutive generated rows, mirroring the template cadence. */
const ROW_INTERVAL_MS = 23

/**
 * Mocked server-side fetch. Generates a block of rows from the template array,
 * simulating network latency. Swap this for a real `/operations` endpoint call;
 * the signature already matches `DataTableServerProps.fetchRows`.
 */
export const fetchOperations = ({
    startRow,
    endRow,
}: {
    startRow: number
    endRow: number
}): Promise<{ rows: Operation[]; total: number }> =>
    new Promise((resolve) => {
        setTimeout(() => {
            const count = Math.max(0, Math.min(endRow, TOTAL_ROWS) - startRow)
            const rows: Operation[] = Array.from({ length: count }, (_, i) => {
                const index = startRow + i
                const template = operations[index % operations.length]
                // Keep `retries` from the template so some rows are expandable.
                return { ...template, id: String(index), time: new Date(BASE_TIME - index * ROW_INTERVAL_MS) }
            })
            resolve({ rows, total: TOTAL_ROWS })
        }, 350)
    })
