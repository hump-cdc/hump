import type { FC } from 'react'

import { AreaChartCard } from '@/shared/ui/card'
import { formatNumber2 } from '@/shared/utils'

import { useReplicationLag } from '../model/use-replication-lag'

const formatSeconds = (value: number): string => formatNumber2(value, 's')

export const ReplicationLag: FC = () => {
    const { values } = useReplicationLag()

    return (
        <AreaChartCard
            title="Replication lag"
            subtitle="sec"
            hint="Commit lag is the delay between a change being saved in the database and the pipeline picking it up. Sender lag is the time a change spends travelling from the database to the pipeline."
            data={values}
            series={[
                { dataKey: 'commit', label: 'commit', variant: 'primary' },
                { dataKey: 'sender', label: 'sender', variant: 'success' },
            ]}
            xAxisKey="time"
            tooltipValueFormatter={formatSeconds}
        />
    )
}
