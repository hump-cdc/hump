import { useLiveControlsChartStep } from '@/entities/config/state/@x/dashboard'
import { useMetrics } from '@/entities/metrics/@x/dashboard'
import { formatNumber, formatTimeCompact } from '@/shared/utils'

/** A single point on the failed bar chart. */
type Item = { date: string; value: number }

/**
 * Failed-operations card at the chart step from the live controls. `value` is the total
 * across the charted window, not a cumulative all-time count.
 */
export const useFailed = (): { value: string | undefined; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const { data } = useMetrics(chartStep)
    const points = data?.points

    const value = formatNumber(points?.reduce((total, { failed }) => total + failed, 0))
    const values: Item[] = (points ?? []).map(({ timestamp, failed }) => ({
        date: formatTimeCompact(timestamp),
        value: failed,
    }))

    return { value, values }
}
