import { useLiveQuery } from 'dexie-react-hooks'

import { useLiveControlsChartStep } from '@/entities/config/state/@x/dashboard'
import { getScalar } from '@/shared/storage'
import { formatNumber, formatTimeCompact } from '@/shared/utils'

/** A single point on the failed bar chart. */
type Item = { date: string; value: number }

/**
 * Live failed-operations card data, sourced reactively from the `metric_scalar_*`
 * store matching the chart step selected in the live controls. The card value is the
 * total failed across the charted window, not a cumulative all-time count.
 */
export const useFailed = (): { value: string | undefined; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const items = useLiveQuery(() => getScalar('operationsFailed', chartStep), [chartStep])

    // Samples are folded with `sum`, so each stored value is the count that failed during one bucket.
    const value = formatNumber(items?.reduce((total, { value }) => total + value, 0))
    const values: Item[] = (items ?? []).map(({ timestamp, value }) => ({
        date: formatTimeCompact(timestamp),
        value,
    }))

    return { value, values }
}
