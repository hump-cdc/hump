import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [16, 18, 15, 20, 19, 22, 21, 24, 23, 26].map((value, index) => ({ date: `16:0${index}`, value }))

export const Delivered: FC = () => (
    <MetricCard
        title="Delivered"
        tooltip="TODO: describe Delivered"
        value="128 402"
        values={VALUES}
        variant="success"
        aside={null}
    />
)
