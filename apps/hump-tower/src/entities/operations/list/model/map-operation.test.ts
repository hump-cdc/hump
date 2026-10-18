import { describe, expect, test } from 'bun:test'

import type { OperationAttemptModel, OperationItemModel } from '@/shared/api'

import { mapOperation } from './map-operation'

describe('mapOperation', () => {
    test('maps the record fields, converting bigint counters', () => {
        const row = mapOperation(item({ seq: big(42), duration: big(15), operation: 'truncate' }))

        expect(row).toMatchObject({
            id: '42',
            time: new Date('2026-09-22T10:00:00.000Z'),
            table: 'public.users',
            op: 'truncate',
            sink: 'kafka',
            sinkHost: 'localhost:9092',
            target: 'users',
            status: 'delivered',
            attempts: 1,
            durationMs: 15,
            payload: '{}',
            payloadSize: 2,
            metadata: {},
        })
    })

    test('maps a failure with its error', () => {
        const row = mapOperation(item({ status: 'failure', error: 'boom' }))

        expect(row.status).toBe('failed')
        expect(row.error).toBe('boom')
    })

    test('keeps no retries for a single attempt', () => {
        expect(mapOperation(item({ attempts: [attempt(1, 'success')] })).retries).toBeUndefined()
    })

    test('lists every attempt once there was a retry, a retry counting as failed', () => {
        const row = mapOperation(
            item({ retries: big(1), attempts: [attempt(1, 'retry', 'reset'), attempt(2, 'success')] }),
        )

        expect(row.attempts).toBe(2)
        expect(row.retries?.map((a) => [a.attempt, a.result, a.error, a.durationMs])).toEqual([
            [1, 'failed', 'reset', 3],
            [2, 'delivered', undefined, 3],
        ])
    })

    test('falls back to the retry count when attempts are missing', () => {
        const row = mapOperation(item({ attempts: null, retries: big(2) }))

        expect(row.attempts).toBe(3)
        expect(row.retries).toBeUndefined()
    })
})

/** int64 fields arrive as bigint after the response transformer, though the types say number. */
const big = (value: number): number => BigInt(value) as unknown as number

const attempt = (index: number, status: string, error?: string): OperationAttemptModel => ({
    attempt: big(index),
    startedAt: '2026-09-22T10:00:00.000Z',
    duration: big(3),
    status,
    error,
})

const item = (overrides: Partial<OperationItemModel> = {}): OperationItemModel => ({
    seq: big(1),
    startedAt: '2026-09-22T10:00:00.000Z',
    duration: big(5),
    timestamp: '2026-09-22T09:59:59.000Z',
    database: 'hump',
    table: 'public.users',
    operation: 'insert',
    sink: 'kafka',
    host: 'localhost:9092',
    target: 'users',
    status: 'success',
    retries: big(0),
    payload: '{}',
    payloadSize: big(2),
    metadata: {},
    attempts: [attempt(1, 'success')],
    ...overrides,
})
