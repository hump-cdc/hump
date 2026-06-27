import { useLiveQuery } from 'dexie-react-hooks'

import { useLiveControlsChartStep } from '@/entities/config/state/@x/stats'
import { getScalar } from '@/shared/storage'
import { formatTimeCompact } from '@/shared/utils'

/** A single point on the async-queue bar chart. */
type Item = { date: string; value: number }

/**
 * Live async-queue chart data, sourced reactively from the `metric_scalar_*`
 * store matching the chart step selected in the live controls.
 */
export const useAsyncQueue = (): { values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const items = useLiveQuery(() => getScalar('asyncQueue', chartStep), [chartStep])

    const values: Item[] = (items ?? []).map(({ timestamp, value }) => ({
        date: formatTimeCompact(timestamp),
        value,
    }))

    return { values }
}
