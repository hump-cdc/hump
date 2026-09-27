import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { NumberFlow } from '@/shared/ui/number-flow'
import { formatBytes } from '@/shared/utils'

import { useReplicationLag } from '../model/use-replication-lag'

const formatLag = (value: number): string => formatBytes(value)

export const ReplicationLag: FC = () => {
    const { value, values } = useReplicationLag()

    return (
        <MetricCard
            title="Replication lag"
            hint="How far behind the database the pipeline is, measured as the amount of WAL written but not yet picked up. Small values mean you're seeing changes almost as they happen; a growing number means the pipeline can't keep up with the write rate."
            value={value === undefined ? undefined : <NumberFlow value={value} kind="bytes" />}
            values={values}
            tooltipValueFormatter={formatLag}
            variant="primary"
            asideTrend="negative"
        />
    )
}
