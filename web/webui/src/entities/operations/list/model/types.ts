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
    /** Target the sink rendered for the row: topic, subject, exchange, stream or URL. */
    target: string
    status: 'delivered' | 'failed'
    attempts: number
    /** Whole delivery, every attempt and backoff pause included. */
    durationMs: number
    error?: string
    /** Every attempt, oldest first; present only when there was more than one. */
    retries?: Attempt[]
    /** Body as it went on the wire, truncated to the collector's limit. */
    payload: string
    /** Untruncated body size in bytes. */
    payloadSize: number
    /** What the sink reported about the call; headers are keyed under a `header.` prefix. */
    metadata: Record<string, string>
}
