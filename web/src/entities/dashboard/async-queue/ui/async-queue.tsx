import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [12, 9, 14, 11, 16, 13, 10, 15, 12, 17].map((value, index) => ({ date: `16:0${index}`, value }))

export const AsyncQueue: FC = () => (
    <MetricCard
        title="Async queue"
        tooltip="TODO: describe Async queue"
        value="17"
        values={VALUES}
        variant="primary"
        asideTrend="negative"
    />
)
