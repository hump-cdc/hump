import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [14, 10, 12, 9, 13, 8, 11, 7, 10, 6].map((value, index) => ({ date: `16:0${index}`, value }))

export const SenderLag: FC = () => (
    <MetricCard
        title="Sender lag"
        tooltip="TODO: describe Sender lag"
        value="0.4s"
        values={VALUES}
        variant="primary"
        asideTrend="negative"
    />
)
