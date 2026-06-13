import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = Array.from({ length: 100 }, (_, index) => ({
    date: `16:0${index}`,
    value: Math.round(180 + 120 * Math.sin(index / 4) + (index % 7) * 14),
}))

export const AsyncQueueDepth: FC = () => (
    <MetricCard
        title="Async queue depth"
        tooltip="Pending events buffered in the async sender queue."
        values={VALUES}
        variant="primary"
        chartSize="lg"
        asideTrend={null}
    />
)
