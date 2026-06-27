import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { formatNumber2 } from '@/shared/utils'

import { useCommitLag } from '../model/use-commit-lag'

const formatSeconds = (value: number): string => formatNumber2(value, 's')

export const CommitLag: FC = () => {
    const { value, values } = useCommitLag()

    return (
        <MetricCard
            title="Commit lag"
            hint="How far behind the database the pipeline is. It's the time between a change being saved in the database and the pipeline picking it up. Small values mean you're seeing changes almost as they happen; a growing number means the pipeline can't keep up with the write rate."
            value={value}
            values={values}
            tooltipValueFormatter={formatSeconds}
            variant="primary"
            asideTrend="negative"
        />
    )
}
