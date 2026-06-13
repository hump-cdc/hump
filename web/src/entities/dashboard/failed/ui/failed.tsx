import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [6, 8, 7, 10, 9, 7, 11, 8, 6, 12].map((value, index) => ({ date: `16:0${index}`, value }))

export const Failed: FC = () => (
    <MetricCard
        title="Failed"
        tooltip="TODO: describe Failed"
        value="12"
        values={VALUES}
        variant="danger"
        aside={null}
    />
)
