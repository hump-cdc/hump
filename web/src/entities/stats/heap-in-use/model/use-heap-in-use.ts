import { useLiveQuery } from 'dexie-react-hooks'

import { useLiveControlsChartStep } from '@/entities/config/state/@x/stats'
import { getScalar } from '@/shared/storage'
import { formatTimeCompact } from '@/shared/utils'

/** A single point on the heap area chart, in megabytes. */
type Item = { time: string; heapInUse: number }

/** Convert a byte reading to megabytes, rounded to 2 fraction digits. */
const toMegabytes = (bytes: number): number => Math.round((bytes / 1024 / 1024) * 100) / 100

/**
 * Live heap chart data, sourced reactively from the `metric_scalar_*` store matching
 * the chart step selected in the live controls. `value` is the latest reading in
 * megabytes.
 */
export const useHeapInUse = (): { value: number | undefined; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const items = useLiveQuery(() => getScalar('heap', chartStep), [chartStep])

    // `items` is oldest → newest and zero-padded at the front; the last entry is the latest sample.
    const latest = items?.at(-1)?.value
    const value = latest === undefined ? undefined : toMegabytes(latest)
    const values: Item[] = (items ?? []).map(({ timestamp, value }) => ({
        time: formatTimeCompact(timestamp),
        heapInUse: toMegabytes(value),
    }))

    return { value, values }
}
