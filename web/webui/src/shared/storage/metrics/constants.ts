import type { MetricBucket } from './types'

/** Ring-buffer cap: each series keeps at most this many newest samples per bucket store. */
export const METRIC_SAMPLES_MAX = 10

/** Window size in milliseconds of every bucket store. */
export const METRIC_BUCKET_MS: Record<MetricBucket, number> = {
    '1m': 60_000,
    '5m': 300_000,
    '10m': 600_000,
}
