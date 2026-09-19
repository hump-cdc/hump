import { useQuery } from '@tanstack/react-query'

import { Metrics as MetricsApi, type MetricsPointModel, type MetricsResponseBodyModel } from '@/shared/api'
import { keys } from '@/shared/state'

import type { Metrics, MetricsDuration, MetricsPoint } from './types'

/** Metrics refresh every minute, this far past the minute boundary. */
const REFRESH_OFFSET_MS = 1_000

/** Refresh period of the metrics query. */
const REFRESH_PERIOD_MS = 60_000

/** Buckets every series is laid out on, matching the collector's `pointsMax`. */
export const METRICS_POINTS_MAX = 20

/** Metrics of one bucket width, padded to a fixed timeline and repolled at `hh:mm:01` after every fetch. */
export const useMetrics = (duration: MetricsDuration) =>
    useQuery({
        queryKey: keys.queries.metrics(duration),
        queryFn: async ({ signal }) => {
            const { data } = await MetricsApi.get({ body: { duration }, signal, throwOnError: true })
            return data
        },
        select: (data: MetricsResponseBodyModel) => normalizeMetrics(data, Date.now()),
        refetchInterval: () => refetchInterval(),
        staleTime: 10_000,
    })

/** Milliseconds until the next refresh, `REFRESH_OFFSET_MS` past a minute boundary and strictly after `now`. */
export const refetchInterval = (now: number = Date.now()): number => {
    let next = Math.floor(now / REFRESH_PERIOD_MS) * REFRESH_PERIOD_MS + REFRESH_OFFSET_MS
    if (next <= now) next += REFRESH_PERIOD_MS

    return next - now
}

/**
 * Lays the response out on `METRICS_POINTS_MAX` buckets ending at the newest point, so a bucket the
 * collector skipped reads as zero instead of shifting the bars after it. An empty response ends at `now`.
 */
export const normalizeMetrics = (response: MetricsResponseBodyModel, now: number): Metrics => {
    const bucketMs = toNumber(response.bucketSeconds) * 1000
    const points = (response.points ?? []).map(toPoint)
    const byTimestamp = new Map(points.map((point) => [point.timestamp, point]))
    const newest = points.at(-1)?.timestamp ?? Math.floor(now / bucketMs) * bucketMs

    const sinks = new Set<string>()
    for (const point of points) {
        for (const sink of Object.keys(point.sinkDelivery)) sinks.add(sink)
        for (const sink of Object.keys(point.sinkLatencyP95Seconds)) sinks.add(sink)
    }

    return {
        bucketMs,
        points: Array.from({ length: METRICS_POINTS_MAX }, (_, index) => {
            const timestamp = newest - (METRICS_POINTS_MAX - 1 - index) * bucketMs
            return byTimestamp.get(timestamp) ?? emptyPoint(timestamp)
        }),
        sinks: [...sinks].sort(),
    }
}

/** Counters arrive as `bigint` once the response transformer ran, while the generated types say `number`. */
const toNumber = (value: number | bigint): number => Number(value)

/** Converts a per-sink record of counters. */
const toNumbers = (record: Record<string, number | bigint>): Record<string, number> =>
    Object.fromEntries(Object.entries(record).map(([key, value]) => [key, toNumber(value)]))

/** Converts one response point to epoch milliseconds and plain numbers. */
const toPoint = (point: MetricsPointModel): MetricsPoint => ({
    timestamp: Date.parse(point.timestamp),
    delivered: toNumber(point.delivered),
    failed: toNumber(point.failed),
    throughput: point.throughput,
    replicationLagBytes: point.replicationLagBytes,
    memoryAllocBytes: point.memoryAllocBytes,
    sinkDelivery: toNumbers(point.sinkDelivery),
    sinkLatencyP95Seconds: toNumbers(point.sinkLatencyP95Seconds),
})

/** All-zero point standing in for a bucket the response has no data for. */
const emptyPoint = (timestamp: number): MetricsPoint => ({
    timestamp,
    delivered: 0,
    failed: 0,
    throughput: 0,
    replicationLagBytes: 0,
    memoryAllocBytes: 0,
    sinkDelivery: {},
    sinkLatencyP95Seconds: {},
})
