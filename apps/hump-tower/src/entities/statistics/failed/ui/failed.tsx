import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { NumberFlow } from '@/shared/ui/number-flow'

import { useFailed } from '../model/use-failed'

export const Failed: FC = () => {
    const { value, values } = useFailed()

    return (
        <MetricCard
            title="Failed"
            hint="Delivery attempts that failed during the time range."
            value={value === undefined ? undefined : <NumberFlow value={value} kind="integer" />}
            values={values}
            tooltipValue={{ kind: 'integer' }}
            variant="danger"
            aside={null}
        />
    )
}
