import type { FC } from 'react'

import { BarChartCard } from '@/shared/ui/card'

const DATA = Array.from({ length: 16 }, (_, index) => ({
    time: `16:${String(index * 5).padStart(2, '0')}`,
    kafka: 0.3 + ((index * 7) % 20) / 100,
    nats: 0.2 + ((index * 5) % 15) / 100,
    rabbit: 0.15 + ((index * 3) % 12) / 100,
}))

export const SinkLatency: FC = () => (
    <BarChartCard
        title="Sink latency"
        data={DATA}
        series={[
            { dataKey: 'kafka', label: 'kafka', variant: 'primary' },
            { dataKey: 'nats', label: 'nats', variant: 'success' },
            { dataKey: 'rabbit', label: 'rabbit', variant: 'amber' },
        ]}
        xAxisKey="time"
    />
)
