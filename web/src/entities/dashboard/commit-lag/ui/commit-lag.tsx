import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [8, 12, 10, 14, 11, 16, 13, 18, 15, 22].map((value, index) => ({ date: `16:0${index}`, value }))

export const CommitLag: FC = () => (
    <MetricCard
        title="Commit lag"
        tooltip="Commit lag — delay between a row being committed in Postgres and Hump reading it from the replication stream (WAL)."
        value="1.2s"
        values={VALUES}
        variant="primary"
        asideTrend="negative"
    />
)
