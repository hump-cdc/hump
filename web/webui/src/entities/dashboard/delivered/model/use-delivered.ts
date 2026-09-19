import { useLiveControlsChartStep } from '@/entities/config/state/@x/dashboard'
import { useMetrics } from '@/entities/metrics/@x/dashboard'
import { formatNumber, formatTimeCompact } from '@/shared/utils'

/** A single point on the delivered bar chart. */
type Item = { date: string; value: number }

/**
 * Delivered-operations card at the chart step from the live controls. `value` is the total
 * across the charted window, not a cumulative all-time count.
 */
export const useDelivered = (): { value: string | undefined; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const { data } = useMetrics(chartStep)

    const points = data?.points

    const value = formatNumber(points?.reduce((total, { delivered }) => total + delivered, 0))
    const values: Item[] = (points ?? []).map(({ timestamp, delivered }) => ({
        date: formatTimeCompact(timestamp),
        value: delivered,
    }))

    return { value, values }
}
