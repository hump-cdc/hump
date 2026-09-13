import { useLiveQuery } from 'dexie-react-hooks'

import { useLiveControlsChartStep } from '@/entities/config/state/@x/dashboard'
import { getScalar } from '@/shared/storage'
import { formatNumber, formatTimeCompact } from '@/shared/utils'

/** A single point on the async-queue bar chart. */
type Item = { date: string; value: number }

/**
 * Live async-queue card data, sourced reactively from the `metric_scalar_*` store
 * matching the chart step selected in the live controls.
 */
export const useAsyncQueue = (): { value: string | undefined; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const items = useLiveQuery(() => getScalar('asyncQueue', chartStep), [chartStep])

    // `items` is oldest → newest and zero-padded at the front; the last entry is the latest sample.
    const value = formatNumber(items?.at(-1)?.value)
    const values: Item[] = (items ?? []).map(({ timestamp, value }) => ({
        date: formatTimeCompact(timestamp),
        value,
    }))

    return { value, values }
}
