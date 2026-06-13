import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [3, 2, 4, 3, 2, 3, 2, 4, 3, 2].map((value, index) => ({ date: `16:0${index}`, value }))

export const FailRate: FC = () => (
    <MetricCard
        title="Fail rate"
        tooltip="Share of delivery attempts that failed across all sinks."
        value="0.03%"
        values={VALUES}
        variant="primary"
        asideTrend="negative"
    />
)
