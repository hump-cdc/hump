import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [6, 9, 7, 10, 8, 11, 7, 9, 6, 8].map((value, index) => ({ date: `16:0${index}`, value }))

export const SenderLag: FC = () => (
    <MetricCard
        title="Sender lag"
        tooltip="Delay between Hump reading a row and the sink confirming delivery."
        value="0.4s"
        values={VALUES}
        variant="primary"
        asideTrend="negative"
    />
)
