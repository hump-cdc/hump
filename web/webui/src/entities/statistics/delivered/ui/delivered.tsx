import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { NumberFlow } from '@/shared/ui/number-flow'

import { useDelivered } from '../model/use-delivered'

export const Delivered: FC = () => {
    const { value, values } = useDelivered()

    return (
        <MetricCard
            title="Delivered"
            hint="Change events successfully written to their destinations during the time range."
            value={value === undefined ? undefined : <NumberFlow value={value} kind="integer" />}
            values={values}
            tooltipValue={{ kind: 'integer' }}
            variant="success"
            aside={null}
        />
    )
}
