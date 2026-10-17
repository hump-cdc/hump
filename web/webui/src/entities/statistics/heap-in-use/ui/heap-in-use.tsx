import type { FC } from 'react'

import { AreaChartCard } from '@/shared/ui/card'
import { NumberFlow } from '@/shared/ui/number-flow'

import { useHeapInUse } from '../model/use-heap-in-use'

export const HeapInUse: FC = () => {
    const { value, values } = useHeapInUse()

    return (
        <AreaChartCard
            title="Memory"
            hint="Memory the pipelines processing is actively using."
            value={
                value === undefined ? undefined : (
                    <div className="flex flex-nowrap items-end gap-1">
                        <NumberFlow value={value} kind="decimal" />{' '}
                        <span className="pb-1 font-medium font-sans text-base text-ink-subtle">MB</span>
                    </div>
                )
            }
            data={values}
            series={[{ dataKey: 'heapInUse', label: 'heap in use', variant: 'primary' }]}
            xAxisKey="time"
            tooltipValue={{ kind: 'decimal', suffix: ' MB' }}
            chartClassName="h-24"
        />
    )
}
