import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { formatNumber } from '@/shared/utils'

import { useDelivered } from '../model/use-delivered'

const formatDelivered = (value: number): string => formatNumber(value)

export const Delivered: FC = () => {
    const { value, values } = useDelivered()

    return (
        <MetricCard
            title="Delivered"
            hint="Change events successfully written to their destinations during the time range."
            value={value}
            values={values}
            tooltipValueFormatter={formatDelivered}
            variant="success"
            aside={null}
        />
    )
}
