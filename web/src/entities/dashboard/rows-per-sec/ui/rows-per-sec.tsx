import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [10, 13, 11, 15, 14, 18, 16, 20, 19, 24].map((value, index) => ({ date: `16:0${index}`, value }))

export const RowsPerSec: FC = () => (
    <MetricCard title="Rows / sec" tooltip="TODO: describe Rows / sec" value="842" values={VALUES} variant="primary" />
)
