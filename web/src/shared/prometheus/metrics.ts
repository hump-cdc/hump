/**
 * The exporter's families, keyed by the name the code refers to them by. Single source of
 * truth: everything below is derived from it, and every accessor takes a `MetricFamily`, so
 * a mistyped family name is a compile error rather than an empty result at runtime.
 */
export const FAMILY = {
    commitLag: 'hump_lag_commit_seconds',
    senderLag: 'hump_lag_sender_seconds',
    rows: 'hump_processed_rows_duration_seconds_count',
    asyncQueue: 'hump_async_queue_total',
    sinkMessages: 'hump_sink_messages_total',
    sinkDuration: 'hump_sink_duration_seconds_bucket',
    heap: 'go_memstats_sys_bytes',
    processStart: 'process_start_time_seconds',
} as const

/** Any family name {@link FAMILY} declares. */
export type MetricFamily = (typeof FAMILY)[keyof typeof FAMILY]

/** Families kept while parsing; the scrape's hundreds of other lines are dropped. */
export const TRACKED_METRICS: readonly MetricFamily[] = Object.values(FAMILY)

/**
 * Every cumulative family the deltas are derived from. Gauges are excluded: they are stored
 * as read and need no baseline.
 */
export const COUNTER_FAMILIES: readonly MetricFamily[] = [FAMILY.rows, FAMILY.sinkMessages, FAMILY.sinkDuration]

// Delivery outcomes on `hump_sink_messages_total`. The exporter's `retry` is ignored — a
// retried message is counted again under its final outcome.
export const STATUS_SUCCESS = 'success'
export const STATUS_FAILURE = 'failure'
