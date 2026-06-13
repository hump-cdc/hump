import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [8, 12, 10, 14, 11, 9, 13, 10, 12, 8].map((value, index) => ({ date: `16:0${index}`, value }))

export const CommitLag: FC = () => (
    <MetricCard
        title="Commit lag"
        tooltip="Delay between a row being committed in Postgres and Hump reading it from the WAL."
        value="1.2s"
        values={VALUES}
        variant="primary"
        asideTrend="negative"
    />
)
