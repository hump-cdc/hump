import type { FC } from 'react'

import { AreaChartCard } from '@/shared/ui/card'
import { formatNumberCompact } from '@/shared/utils'

import { useThroughput } from '../model/use-throughput'

const formatRate = (value: number): string => formatNumberCompact(value)

export const Throughput: FC = () => {
    const { values } = useThroughput()

    return (
        <AreaChartCard
            title="Throughput"
            subtitle="rows/sec"
            hint="Row changes processed per second."
            data={values}
            series={[{ dataKey: 'rowsPerSec', label: 'rows/sec', variant: 'primary' }]}
            xAxisKey="time"
            tooltipValueFormatter={formatRate}
        />
    )
}
