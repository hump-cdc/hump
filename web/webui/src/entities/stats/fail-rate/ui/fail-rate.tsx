import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { formatNumber2 } from '@/shared/utils'

import { useFailRate } from '../model/use-fail-rate'

const formatPercent = (value: number): string => formatNumber2(value, '%')

export const FailRate: FC = () => {
    const { value, values } = useFailRate()

    return (
        <MetricCard
            title="Fail rate"
            hint="Failed deliveries as a percentage of successful ones, across all sinks."
            value={formatNumber2(value, '%')}
            values={values}
            tooltipValueFormatter={formatPercent}
            variant="danger"
            separated
            chartClassName="h-24"
            asideTrend="negative"
        />
    )
}
