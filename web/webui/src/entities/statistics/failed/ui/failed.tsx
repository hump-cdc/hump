import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { NumberFlow } from '@/shared/ui/number-flow'
import { formatNumber } from '@/shared/utils'

import { useFailed } from '../model/use-failed'

const formatFailed = (value: number): string => formatNumber(value)

export const Failed: FC = () => {
    const { value, values } = useFailed()

    return (
        <MetricCard
            title="Failed"
            hint="Delivery attempts that failed during the time range."
            value={value === undefined ? undefined : <NumberFlow value={value} kind="integer" />}
            values={values}
            tooltipValueFormatter={formatFailed}
            variant="danger"
            aside={null}
        />
    )
}
