/** One attempt of a delivery; a `retry` attempt counts as failed. */
export type Attempt = {
    attempt: number
    started: Date
    durationMs: number
    result: 'failed' | 'delivered'
    error?: string
}

/** One delivery of one row to one sink, as the operations list shows it. */
export type Operation = {
    /** The record's `seq`, unique across the process. */
    id: string
    /** When the first attempt began. */
    time: Date
    /** Schema-qualified source table. */
    table: string
    op: 'insert' | 'update' | 'delete' | 'truncate'
    /** Sink type. */
    sink: string
    sinkHost: string
    status: 'delivered' | 'failed'
    attempts: number
    /** Whole delivery, every attempt and backoff pause included. */
    durationMs: number
    error?: string
    /** Every attempt, oldest first; present only when there was more than one. */
    retries?: Attempt[]
}
