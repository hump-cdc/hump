import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { formatNumber } from '@/shared/utils'

import { useAsyncQueue } from '../model/use-async-queue'

const formatDepth = (value: number): string => formatNumber(value)

export const AsyncQueue: FC = () => {
    const { value, values } = useAsyncQueue()

    return (
        <MetricCard
            title="Async queue"
            hint="Pending events buffered in the async delivery pipeline."
            value={value}
            values={values}
            tooltipValueFormatter={formatDepth}
            variant="primary"
            asideTrend="negative"
        />
    )
}
