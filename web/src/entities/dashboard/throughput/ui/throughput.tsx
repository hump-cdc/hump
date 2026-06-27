import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { formatNumberCompact } from '@/shared/utils'

import { useThroughput } from '../model/use-throughput'

const formatRate = (value: number): string => formatNumberCompact(value)

export const Throughput: FC = () => {
    const { value, values } = useThroughput()

    return (
        <MetricCard
            title="Throughput"
            hint="Row changes processed per second."
            value={value}
            values={values}
            tooltipValueFormatter={formatRate}
            variant="primary"
        />
    )
}
