import type { FC } from 'react'

import { BarChartCard } from '@/shared/ui/card'
import { formatNumber2 } from '@/shared/utils'

import { useSinkLatency } from '../model/use-sink-latency'

const formatMs = (value: number): string => formatNumber2(value, 'ms')

export const SinkLatency: FC = () => {
    const { series, values } = useSinkLatency()

    return (
        <BarChartCard
            title="Sink latency"
            subtitle="P95(ms)"
            hint="Time spent writing one change event to its destination, by sink type. P95 means 95% of writes finished faster than this."
            data={values}
            series={series}
            xAxisKey="time"
            stacked={true}
            tooltipValueFormatter={formatMs}
        />
    )
}
