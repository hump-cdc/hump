import type { FC } from 'react'

import { AreaChartCard } from '@/shared/ui/card'

import { useThroughput } from '../model/use-throughput'

export const Throughput: FC = () => {
    const { values } = useThroughput()

    return (
        <AreaChartCard
            title="Throughput"
            subtitle="events/sec"
            hint="Change events successfully delivered per second."
            data={values}
            series={[{ dataKey: 'eventsPerSec', label: 'events/sec', variant: 'primary' }]}
            xAxisKey="time"
            tooltipValue={{ kind: 'compact' }}
        />
    )
}
