import { useLiveQuery } from 'dexie-react-hooks'

import { useLiveControlsChartStep } from '@/entities/config/state/@x/stats'
import { getScalar } from '@/shared/storage'
import { formatTimeCompact } from '@/shared/utils'

/** A single point on the replication-lag area chart, in seconds. */
type Item = { time: string; commit: number; sender: number }

/**
 * Live replication-lag chart data — commit and sender lag on one timeline — sourced
 * reactively from the `metric_scalar_*` store matching the chart step selected in the
 * live controls.
 */
export const useReplicationLag = (): { values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const items = useLiveQuery(
        async () => ({
            commit: await getScalar('commitLag', chartStep),
            sender: await getScalar('senderLag', chartStep),
        }),
        [chartStep],
    )

    // Both series are written by the same scrape, so the commit series drives the buckets
    // and sender values are looked up by timestamp; a bucket the sender series never
    // reached reads as 0.
    const senderByBucket = new Map((items?.sender ?? []).map(({ timestamp, value }) => [timestamp, value]))
    const values: Item[] = (items?.commit ?? []).map(({ timestamp, value }) => ({
        time: formatTimeCompact(timestamp),
        commit: value,
        sender: senderByBucket.get(timestamp) ?? 0,
    }))

    return { values }
}
