import type { FC } from 'react'

import { BarChartCard } from '@/shared/ui/card'

import { useSinkLatency } from '../model/use-sink-latency'

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
            tooltipValue={{ kind: 'decimal', suffix: 'ms' }}
        />
    )
}
