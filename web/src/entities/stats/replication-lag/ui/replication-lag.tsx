import type { FC } from 'react'

import { AreaChartCard } from '@/shared/ui/card'

const COMMIT = [1.4, 1.1, 1.6, 1.3, 1.8, 1.5, 1.9, 1.6, 2.1, 1.7, 1.5, 1.2]
const SENDER = [0.5, 0.4, 0.6, 0.5, 0.7, 0.5, 0.8, 0.6, 0.7, 0.5, 0.4, 0.4]

const DATA = COMMIT.map((commit, index) => ({
    time: `16:${String(index * 5).padStart(2, '0')}`,
    commit,
    sender: SENDER[index],
}))

export const ReplicationLag: FC = () => (
    <AreaChartCard
        title="Replication lag"
        data={DATA}
        series={[
            { dataKey: 'commit', label: 'commit', variant: 'primary' },
            { dataKey: 'sender', label: 'sender', variant: 'success' },
        ]}
        xAxisKey="time"
    />
)
