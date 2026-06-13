import type { FC } from 'react'

import { AreaChartCard } from '@/shared/ui/card'

const ROWS = [620, 700, 660, 780, 720, 840, 800, 910, 860, 980, 920, 1040]

const DATA = ROWS.map((rowsPerSec, index) => ({
    time: `16:${String(index * 5).padStart(2, '0')}`,
    rowsPerSec,
}))

export const Throughput: FC = () => (
    <AreaChartCard
        title="Throughput"
        subtitle="rows/sec"
        data={DATA}
        series={[{ dataKey: 'rowsPerSec', label: 'rows/sec', variant: 'primary' }]}
        xAxisKey="time"
    />
)
