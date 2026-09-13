import { useLiveQuery } from 'dexie-react-hooks'

import { useLiveControlsChartStep } from '@/entities/config/state/@x/dashboard'
import { getScalar } from '@/shared/storage'
import { formatNumber2, formatTimeCompact } from '@/shared/utils'

/** A single point on the replication-lag bar chart. */
type Item = { date: string; value: number }

/**
 * Live replication-lag card data, sourced reactively from the `metric_scalar_*` store
 * matching the chart step selected in the live controls.
 */
export const useReplicationLag = (): { value: string | undefined; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const items = useLiveQuery(() => getScalar('commitLag', chartStep), [chartStep])

    // `items` is oldest → newest and zero-padded at the front; the last entry is the latest sample.
    const value = formatNumber2(items?.at(-1)?.value, 's')
    const values: Item[] = (items ?? []).map(({ timestamp, value }) => ({
        date: formatTimeCompact(timestamp),
        value,
    }))

    return { value, values }
}
