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
    sinkLatencyP95Seconds: Record<string, number>
}

/** Metrics laid out on a fixed timeline of `METRICS_POINTS_MAX` buckets, oldest first. */
export type Metrics = {
    bucketMs: number
    points: MetricsPoint[]
    /** Every sink type seen across the points, sorted by name. */
    sinks: string[]
}
