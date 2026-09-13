import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { formatNumber2 } from '@/shared/utils'

import { useSenderLag } from '../model/use-sender-lag'

const formatSeconds = (value: number): string => formatNumber2(value, 's')

export const SenderLag: FC = () => {
    const { value, values } = useSenderLag()

    return (
        <MetricCard
            title="Sender lag"
            hint="How long a change spends in transit. It's the time between the database handing a change off and the pipeline receiving it. This covers the network and the database's own work preparing the change, so it stays low on a healthy connection and rises when the link is slow or the database is under strain."
            value={value}
            values={values}
            tooltipValueFormatter={formatSeconds}
            variant="primary"
            asideTrend="negative"
        />
    )
}
