import { useLiveQuery } from 'dexie-react-hooks'

import { useLiveControlsChartStep } from '@/entities/config/state/@x/stats'
import { getSinkDelivery, METRIC_BUCKET_MS, METRIC_SAMPLES_MAX } from '@/shared/storage'
import { formatTimeCompact } from '@/shared/utils'

/** Bar colours, cycled when a scrape carries more sink types than there are variants. */
const VARIANTS = ['primary', 'success', 'amber', 'danger'] as const

/** One charted sink: its samples keyed by sink type, which doubles as the legend label. */
type Series = {
    dataKey: string
    label: string
    variant: (typeof VARIANTS)[number]
}

/** A single point on the sink-delivery bar chart: a bucket label plus one count per sink. */
type Item = Record<string, number | string> & { time: string }

/** Value lookup key of one sink's sample. Sink types carry no spaces, so the join is unambiguous. */
const sampleKey = (sink: string, timestamp: number): string => `${sink} ${timestamp}`

/**
 * Live sink-delivery chart data, sourced reactively from the `metric_sink_delivery_*`
 * store matching the chart step selected in the live controls. One series per sink type,
 * derived from the sinks the store actually carries.
 */
export const useSinkDelivery = (): { series: Series[]; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const bucketMs = METRIC_BUCKET_MS[chartStep]
    const items = useLiveQuery(() => getSinkDelivery(chartStep), [chartStep]) ?? []

    // Sorting by name keeps both the stacking order and each sink's colour stable as sinks
    // come and go between polls.
    const series: Series[] = [...new Set(items.map(({ sink }) => sink))]
        .sort()
        .map((sink, index) => ({ dataKey: sink, label: sink, variant: VARIANTS[index % VARIANTS.length] }))

    // Every sink is laid out on one shared timeline of `METRIC_SAMPLES_MAX` buckets ending at
    // the newest sample seen, so a sink that stayed idle for a window reads as 0 there instead
    // of shifting its remaining bars onto the wrong buckets. An empty store falls back to the
    // current bucket, which renders an all-zero chart.
    const sinks = series.map(({ dataKey }) => dataKey)
    const valueByKey = new Map(items.map(({ sink, timestamp, value }) => [sampleKey(sink, timestamp), value]))
    const newest = items.reduce(
        (latest, { timestamp }) => Math.max(latest, timestamp),
        items.length === 0 ? Math.floor(Date.now() / bucketMs) * bucketMs : 0,
    )

    return {
        series,
        values: Array.from({ length: METRIC_SAMPLES_MAX }, (_, index) => {
            const timestamp = newest - (METRIC_SAMPLES_MAX - 1 - index) * bucketMs
            const counts = sinks.map((sink) => [sink, valueByKey.get(sampleKey(sink, timestamp)) ?? 0] as const)

            return { time: formatTimeCompact(timestamp), ...Object.fromEntries(counts) }
        }),
    }
}
