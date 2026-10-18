import { useLiveControlsChartStep } from '@/entities/config/state/@x/statistics'
import { useMetrics } from '@/entities/metrics/@x/statistics'
import { formatTimeCompact } from '@/shared/utils'

/** Bar colours, cycled when the window carries more sink types than there are variants. */
const VARIANTS = ['primary', 'success', 'amber', 'danger'] as const

/** One charted sink: its samples keyed by sink type, which doubles as the legend label. */
type Series = {
    dataKey: string
    label: string
    variant: (typeof VARIANTS)[number]
}

/** A single point on the sink-latency bar chart: a bucket label plus one p95 per sink, in milliseconds. */
type Item = Record<string, number | string> & { time: string }

/** Round a latency in milliseconds to 2 fraction digits. */
const round = (milliseconds: number): number => Math.round(milliseconds * 100) / 100

/**
 * Sink-latency chart at the chart step from the live controls. One series per sink type seen in
 * the window; a bucket without an estimate for a sink reads as 0.
 */
export const useSinkLatency = (): { series: Series[]; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const { data } = useMetrics(chartStep)
    const sinks = data?.sinks ?? []

    // Sinks come sorted by name, which keeps the stacking order and each sink's colour stable as sinks come and go.
    const series: Series[] = sinks.map((sink, index) => ({
        dataKey: sink,
        label: sink,
        variant: VARIANTS[index % VARIANTS.length],
    }))

    return {
        series,
        values: (data?.points ?? []).map(({ timestamp, sinkLatencyP95 }) => ({
            time: formatTimeCompact(timestamp),
            ...Object.fromEntries(sinks.map((sink) => [sink, round(sinkLatencyP95[sink] ?? 0)])),
        })),
    }
}
