import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { formatNumber2 } from '@/shared/utils'

import { useReplicationLag } from '../model/use-replication-lag'

const formatSeconds = (value: number): string => formatNumber2(value, 's')

export const ReplicationLag: FC = () => {
    const { value, values } = useReplicationLag()

    return (
        <MetricCard
            title="Replication lag"
            hint="How far behind the database the pipeline is. It's the time between a change being saved in the database and the pipeline picking it up. Small values mean you're seeing changes almost as they happen; a growing number means the pipeline can't keep up with the write rate."
            value={value}
            values={values}
            tooltipValueFormatter={formatSeconds}
            variant="primary"
            asideTrend="negative"
        />
    )
}
