import { useLiveControlsChartStep } from '@/entities/config/state/@x/dashboard'
import { useMetrics } from '@/entities/metrics/@x/dashboard'
import { formatTimeCompact } from '@/shared/utils'

/** Bar colours, cycled when the window carries more sink types than there are variants. */
const VARIANTS = ['primary', 'success', 'amber', 'danger'] as const

/** One charted sink: its samples keyed by sink type, which doubles as the legend label. */
type Series = {
    dataKey: string
    label: string
    variant: (typeof VARIANTS)[number]
}

/** A single point on the sink-delivery bar chart: a bucket label plus one count per sink. */
type Item = Record<string, number | string> & { time: string }

/**
 * Sink-delivery chart at the chart step from the live controls. One series per sink type seen
 * in the window; a sink idle in a bucket reads as 0 there.
 */
export const useSinkDelivery = (): { series: Series[]; values: Item[] } => {
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
        values: (data?.points ?? []).map(({ timestamp, sinkDelivery }) => ({
            time: formatTimeCompact(timestamp),
            ...Object.fromEntries(sinks.map((sink) => [sink, sinkDelivery[sink] ?? 0])),
        })),
    }
}
