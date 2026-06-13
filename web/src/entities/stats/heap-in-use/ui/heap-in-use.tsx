import type { FC } from 'react'

import { Cpu } from 'lucide-react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [12, 16, 14, 19, 17, 21, 19, 22].map((value, index) => ({ date: `16:0${index}`, value }))

export const HeapInUse: FC = () => (
    <MetricCard
        title="Heap in use"
        value={
            <div className="flex flex-nowrap items-end gap-1">
                412 <span className="pb-1 font-sans text-base font-medium text-ink-subtle">MB</span>
            </div>
        }
        values={VALUES}
        variant="primary"
        aside={<Cpu className="size-4 text-primary" />}
    />
)
