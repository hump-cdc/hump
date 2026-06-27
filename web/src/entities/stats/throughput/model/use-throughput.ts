import { useLiveQuery } from 'dexie-react-hooks'

import { useLiveControlsChartStep } from '@/entities/config/state/@x/stats'
import { getScalar, METRIC_BUCKET_MS } from '@/shared/storage'
import { formatTimeCompact } from '@/shared/utils'

/** A single point on the throughput area chart. */
type Item = { time: string; rowsPerSec: number }

/**
 * Convert a `bucketMs`-wide bucket row count to a rows-per-second rate, rounded to
 * 2 decimals. Samples are folded with `sum`, so a stored value is the number of rows
 * processed during the whole bucket.
 */
const toRate = (value: number, bucketMs: number): number => Math.round((value / (bucketMs / 1000)) * 100) / 100

/**
 * Live throughput chart data, sourced reactively from the `metric_scalar_*` store
 * matching the chart step selected in the live controls.
 */
export const useThroughput = (): { values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const bucketMs = METRIC_BUCKET_MS[chartStep]
    const items = useLiveQuery(() => getScalar('rows', chartStep), [chartStep])

    const values: Item[] = (items ?? []).map(({ timestamp, value }) => ({
        time: formatTimeCompact(timestamp),
        rowsPerSec: toRate(value, bucketMs),
    }))

    return { values }
}
