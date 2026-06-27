import type { Table } from 'dexie'

/** Bucket window a metric series is stored under (`metric_*_1m` / `_5m` / `_10m`). */
export type MetricBucket = '1m' | '5m' | '10m'

/** A bucket store paired with its window size in milliseconds. */
export type Bucket<T, TKey> = { table: () => Table<T, TKey>; sizeMs: number }

/** A single bucketed metric observation. The rounded `timestamp` is the primary key. */
export type MetricDto = {
    timestamp: number
    value: number
}

/** Names of the single-value metric series stored in the `metric_scalar_*` stores. */
export type ScalarMetricName =
    | 'commitLag'
    | 'senderLag'
    | 'rows'
    | 'asyncQueue'
    | 'operationsDelivered'
    | 'operationsFailed'
    | 'failRate'
    | 'heap'

/** A bucketed observation of one scalar series. The `[metric+timestamp]` pair is the primary key. */
export type ScalarMetricDto = MetricDto & {
    metric: ScalarMetricName
}

/** A bucketed observation scoped to one sink. The `[sink+timestamp]` pair is the primary key. */
export type SinkMetricDto = MetricDto & {
    sink: string
}
