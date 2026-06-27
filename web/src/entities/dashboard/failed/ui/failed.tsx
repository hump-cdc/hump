import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { formatNumber } from '@/shared/utils'

import { useFailed } from '../model/use-failed'

const formatFailed = (value: number): string => formatNumber(value)

export const Failed: FC = () => {
    const { value, values } = useFailed()

    return (
        <MetricCard
            title="Failed"
            hint="Delivery attempts that failed during the time range."
            value={value}
            values={values}
            tooltipValueFormatter={formatFailed}
            variant="danger"
            aside={null}
        />
    )
}
