import { useLiveQuery } from 'dexie-react-hooks'

import { useLiveControlsChartStep } from '@/entities/config/state/@x/stats'
import { getScalar } from '@/shared/storage'
import { formatTimeCompact } from '@/shared/utils'

/** A single point on the fail-rate bar chart, in percent. */
type Item = { date: string; value: number }

/**
 * Live fail-rate chart data, sourced reactively from the `metric_scalar_*` store
 * matching the chart step selected in the live controls. `value` is the latest
 * reading in percent.
 */
export const useFailRate = (): { value: number | undefined; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const items = useLiveQuery(() => getScalar('failRate', chartStep), [chartStep])

    // `items` is oldest → newest and zero-padded at the front; the last entry is the latest sample.
    const value = items?.at(-1)?.value
    const values: Item[] = (items ?? []).map(({ timestamp, value }) => ({
        date: formatTimeCompact(timestamp),
        value,
    }))

    return { value, values }
}
