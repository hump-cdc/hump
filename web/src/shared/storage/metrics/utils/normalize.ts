import { METRIC_SAMPLES_MAX } from '../constants'
import type { MetricDto } from '../types'

/**
 * Normalize a bucketed series for charting: the newest `size` samples, oldest → newest, prepended
 * with zero-valued `bucketMs` buckets so a chart always renders `size` points.
 */
export const normalize = (samples: MetricDto[], bucketMs: number, size: number = METRIC_SAMPLES_MAX): MetricDto[] => {
    const sorted = [...samples]
        .sort((a, b) => a.timestamp - b.timestamp)
        .slice(-size)
        .map(({ timestamp, value }) => ({ timestamp, value }))

    const earliest = sorted[0]?.timestamp ?? Math.floor(Date.now() / bucketMs) * bucketMs
    const padding: MetricDto[] = []
    for (let i = sorted.length; i < size; i++) {
        padding.push({ timestamp: earliest - (size - i) * bucketMs, value: 0 })
    }

    return [...padding, ...sorted]
}
