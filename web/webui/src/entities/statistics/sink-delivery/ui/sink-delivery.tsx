import type { FC } from 'react'

import { BarChartCard } from '@/shared/ui/card'
import { formatNumber } from '@/shared/utils'

import { useSinkDelivery } from '../model/use-sink-delivery'

const formatDelivered = (value: number): string => formatNumber(value)

export const SinkDelivery: FC = () => {
    const { series, values } = useSinkDelivery()

    return (
        <BarChartCard
            title="Sink delivery"
            hint="Change events delivered to their destinations by sink type, including failed attempts."
            data={values}
            series={series}
            xAxisKey="time"
            tooltipValueFormatter={formatDelivered}
        />
    )
}
