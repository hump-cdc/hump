import { describe, expect, test } from 'bun:test'

import type { MetricsPointModel, MetricsResponseBodyModel } from '@/shared/api'

import { METRICS_POINTS_MAX, normalizeMetrics, refetchInterval } from './use-metrics'

const MINUTE = 60_000
const NOW = Date.UTC(2026, 0, 1, 12, 30, 15)
const CURRENT = Date.UTC(2026, 0, 1, 12, 30)

const at = (minutes: number, seconds: number, ms = 0): number => Date.UTC(2026, 0, 1, 12, minutes, seconds, ms)

const point = (timestamp: number, overrides: Partial<MetricsPointModel> = {}): MetricsPointModel => ({
    timestamp: new Date(timestamp).toISOString(),
    delivered: 0,
    failed: 0,
    throughput: 0,
    replicationLagBytes: 0,
    memoryAllocBytes: 0,
    sinkDelivery: {},
    sinkLatencyP95: {},
    ...overrides,
})

const response = (overrides: Partial<MetricsResponseBodyModel> = {}): MetricsResponseBodyModel => ({
    duration: '1m',
    bucketSeconds: 60,
    points: null,
    tables: {},
    sinks: [],
    connections: [],
    ...overrides,
})

describe('refetchInterval', () => {
    test('waits for :01 of the same minute', () => {
        expect(refetchInterval(at(0, 0, 0))).toBe(1_000)
        expect(refetchInterval(at(0, 0, 999))).toBe(1)
    })

    test('skips to the next minute exactly at :01', () => {
        expect(refetchInterval(at(0, 1, 0))).toBe(60_000)
    })

    test('crosses the minute boundary', () => {
        expect(refetchInterval(at(0, 59, 500))).toBe(1_500)
    })
})

describe('normalizeMetrics', () => {
    test('pads an empty response to zeros ending at the current bucket', () => {
        const metrics = normalizeMetrics(response(), NOW)

        expect(metrics.bucketMs).toBe(MINUTE)
        expect(metrics.sinks).toEqual([])
        expect(metrics.points).toHaveLength(METRICS_POINTS_MAX)
        expect(metrics.points.at(-1)?.timestamp).toBe(CURRENT)
        expect(metrics.points[0].timestamp).toBe(CURRENT - (METRICS_POINTS_MAX - 1) * MINUTE)
        expect(metrics.points.every(({ delivered }) => delivered === 0)).toBe(true)
    })

    test('keeps skipped buckets as zeros on the timeline', () => {
        const metrics = normalizeMetrics(
            response({ points: [point(CURRENT - 2 * MINUTE, { delivered: 5 }), point(CURRENT, { delivered: 7 })] }),
            NOW,
        )

        expect(metrics.points.slice(-3).map(({ delivered }) => delivered)).toEqual([5, 0, 7])
    })

    test('converts bigint counters and collects sinks sorted', () => {
        const metrics = normalizeMetrics(
            response({
                bucketSeconds: 60n as unknown as number,
                points: [
                    point(CURRENT, {
                        delivered: 3n as unknown as number,
                        sinkDelivery: { webhook: 2n as unknown as number },
                        sinkLatencyP95: { kafka: 500 },
                    }),
                ],
            }),
            NOW,
        )

        const latest = metrics.points.at(-1)
        expect(latest?.delivered).toBe(3)
        expect(latest?.sinkDelivery).toEqual({ webhook: 2 })
        expect(metrics.sinks).toEqual(['kafka', 'webhook'])
    })
})
