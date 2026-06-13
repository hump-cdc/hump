import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [10, 14, 11, 16, 13, 18, 15, 20, 17, 22].map((value, index) => ({ date: `16:0${index}`, value }))

export const RowsPerSec: FC = () => (
    <MetricCard
        title="Rows / sec"
        tooltip="Replicated rows read from the WAL per second."
        value="842"
        values={VALUES}
        variant="primary"
        asideTrend="positive"
    />
)
