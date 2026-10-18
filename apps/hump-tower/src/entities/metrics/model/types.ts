import type { MetricsRequestBodyModel } from '@/shared/api'

/** Width of the buckets the metrics are aggregated into. */
export type MetricsDuration = MetricsRequestBodyModel['duration']

/** One bucket of metrics, counters as plain numbers and the bucket start in epoch milliseconds. */
export type MetricsPoint = {
    timestamp: number
    delivered: number
    failed: number
    throughput: number
    replicationLagBytes: number
    memoryAllocBytes: number
    sinkDelivery: Record<string, number>
    sinkLatencyP95: Record<string, number>
}

/** Row operations replicated from one table, by operation. */
export type MetricsTable = {
    /** Schema-qualified table name. */
    name: string
    insert: MetricsTableOp
    update: MetricsTableOp
    delete: MetricsTableOp
    truncate: MetricsTableOp
}

/** One operation kind of a table. */
export type MetricsTableOp = {
    /** Operations replicated since start. */
    count: number
    /** P95 row processing latency in milliseconds over the requested window; absent without an estimate. */
    p95?: number
}

/** What a sink, or one of its targets, delivered since start. */
export type MetricsDelivery = {
    operations: number
    /** Share of deliveries that succeeded, 0 to 1; absent without an operation. */
    success?: number
    /** P95 write latency in milliseconds over the requested window; absent without an estimate. */
    p95?: number
}

/** One rendered destination of a sink: topic, subject, exchange, stream or URL. */
export type MetricsSinkTarget = MetricsDelivery & {
    target: string
}

/** One sink endpoint, identified by its type and host, with its targets sorted by name. */
export type MetricsSink = MetricsDelivery & {
    type: string
    host: string
    targets: MetricsSinkTarget[]
}

/** One table-to-target edge of the pipeline with the deliveries that flowed along it. */
export type MetricsConnection = {
    /** Schema-qualified table name. */
    table: string
    /** Sink type. */
    sink: string
    host: string
    target: string
    /** Successful deliveries over the connection since start. */
    success: number
    /** Failed deliveries over the connection since start, retries excluded. */
    failure: number
}

/** Configured pipeline: tables, sinks and the connections between them, each sorted for a stable layout. */
export type MetricsTopology = {
    tables: MetricsTable[]
    sinks: MetricsSink[]
    connections: MetricsConnection[]
}

/** Metrics laid out on a fixed timeline of `METRICS_POINTS_MAX` buckets, oldest first. */
export type Metrics = {
    bucketMs: number
    points: MetricsPoint[]
    /** Every sink type seen across the points, sorted by name. */
    sinks: string[]
    topology: MetricsTopology
}
