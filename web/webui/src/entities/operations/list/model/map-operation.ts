import type { OperationAttemptModel, OperationItemModel } from '@/shared/api'

import type { Attempt, Operation } from './types'

/** Converts one API record to a list row. */
export const mapOperation = (item: OperationItemModel): Operation => {
    const attempts = (item.attempts ?? []).map(mapAttempt)

    return {
        id: String(item.seq),
        time: new Date(item.startedAt),
        table: item.table,
        op: item.operation as Operation['op'],
        sink: item.sink,
        sinkHost: item.host,
        target: item.target,
        status: item.status === 'success' ? 'delivered' : 'failed',
        attempts: attempts.length || toNumber(item.retries) + 1,
        durationMs: toNumber(item.duration),
        error: item.error,
        retries: attempts.length > 1 ? attempts : undefined,
        payload: item.payload,
        payloadSize: toNumber(item.payloadSize),
        metadata: item.metadata ?? {},
    }
}

/** Converts one attempt; only a `success` counts as delivered. */
const mapAttempt = (attempt: OperationAttemptModel): Attempt => ({
    attempt: toNumber(attempt.attempt),
    started: new Date(attempt.startedAt),
    durationMs: toNumber(attempt.duration),
    result: attempt.status === 'success' ? 'delivered' : 'failed',
    error: attempt.error,
})

/** int64 fields arrive as `bigint` once the response transformer ran, while the generated types say `number`. */
const toNumber = (value: number | bigint): number => Number(value)
