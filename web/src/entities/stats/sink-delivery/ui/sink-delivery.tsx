import type { FC } from 'react'

import { BarChartCard } from '@/shared/ui/card'

const DATA = Array.from({ length: 16 }, (_, index) => ({
    time: `16:${String(index * 5).padStart(2, '0')}`,
    kafka: 120 + ((index * 17) % 60),
    nats: 80 + ((index * 11) % 40),
    rabbit: 60 + ((index * 7) % 30),
    fail: index % 5 === 0 ? 12 : 4,
}))

export const SinkDelivery: FC = () => (
    <BarChartCard
        title="Sink delivery"
        subtitle="msg/sec"
        data={DATA}
        series={[
            { dataKey: 'kafka', label: 'kafka', variant: 'primary' },
            { dataKey: 'nats', label: 'nats', variant: 'success' },
            { dataKey: 'rabbit', label: 'rabbit', variant: 'amber' },
            { dataKey: 'fail', label: 'fail', variant: 'danger' },
        ]}
        xAxisKey="time"
    />
)
