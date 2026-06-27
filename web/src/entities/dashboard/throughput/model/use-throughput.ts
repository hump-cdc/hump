import { useLiveQuery } from 'dexie-react-hooks'

import { useLiveControlsChartStep } from '@/entities/config/state/@x/dashboard'
import { getScalar, METRIC_BUCKET_MS } from '@/shared/storage'
import { formatNumberCompact, formatTimeCompact } from '@/shared/utils'

/** A single point on the throughput bar chart. */
type Item = { date: string; value: number }

/**
 * Convert a `bucketMs`-wide bucket row count to a rows-per-second rate, rounded to
 * 2 decimals. Samples are folded with `sum`, so a stored value is the number of
 * rows processed during the whole bucket.
 */
const toRate = (value: number, bucketMs: number): number => Math.round((value / (bucketMs / 1000)) * 100) / 100

/**
 * Live throughput card data, sourced reactively from the `metric_scalar_*`
 * store matching the chart step selected in the live controls.
 */
export const useThroughput = (): { value: string | undefined; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const bucketMs = METRIC_BUCKET_MS[chartStep]
    const items = useLiveQuery(() => getScalar('rows', chartStep), [chartStep])

    // `items` is oldest → newest and zero-padded at the front; the last entry is the latest sample.
    const latest = items?.at(-1)?.value
    const value = formatNumberCompact(latest === undefined ? undefined : toRate(latest, bucketMs))
    const values: Item[] = (items ?? []).map(({ timestamp, value }) => ({
        date: formatTimeCompact(timestamp),
        value: toRate(value, bucketMs),
    }))

    return { value, values }
}
