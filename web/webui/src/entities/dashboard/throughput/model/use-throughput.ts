import { useLiveControlsChartStep } from '@/entities/config/state/@x/dashboard'
import { useMetrics } from '@/entities/metrics/@x/dashboard'
import { formatTimeCompact } from '@/shared/utils'

/** A single point on the throughput area chart. */
type Item = { time: string; rowsPerSec: number }

/** Round a rate to 2 fraction digits. */
const round2 = (value: number): number => Math.round(value * 100) / 100

/** Throughput chart at the chart step from the live controls. */
export const useThroughput = (): { values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const { data } = useMetrics(chartStep)

    const values: Item[] = (data?.points ?? []).map(({ timestamp, throughput }) => ({
        time: formatTimeCompact(timestamp),
        rowsPerSec: round2(throughput),
    }))

    return { values }
}
