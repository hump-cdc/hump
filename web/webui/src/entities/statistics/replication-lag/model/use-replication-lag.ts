import { useLiveControlsChartStep } from '@/entities/config/state/@x/statistics'
import { useMetrics } from '@/entities/metrics/@x/statistics'
import { formatBytes, formatTimeCompact } from '@/shared/utils'

/** A single point on the replication-lag bar chart, in bytes. */
type Item = { date: string; value: number }

/** Replication-lag card at the chart step from the live controls. `value` is the latest bucket's lag. */
export const useReplicationLag = (): { value: string | undefined; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const { data } = useMetrics(chartStep)
    const points = data?.points

    // Points run oldest → newest, so the last one is the newest bucket the response reported.
    const value = formatBytes(points?.at(-1)?.replicationLagBytes)
    const values: Item[] = (points ?? []).map(({ timestamp, replicationLagBytes }) => ({
        date: formatTimeCompact(timestamp),
        value: replicationLagBytes,
    }))

    return { value, values }
}
