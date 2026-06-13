import type { FC } from 'react'

import { Activity } from 'lucide-react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [10, 8, 12, 9, 14, 10, 8, 11].map((value, index) => ({ date: `16:0${index}`, value }))

export const GcPause: FC = () => (
    <MetricCard
        title="GC pause"
        value={
            <div className="flex flex-nowrap items-end gap-1">
                1.8 <span className="pb-1 font-sans text-base font-medium text-ink-subtle">ms</span>
            </div>
        }
        values={VALUES}
        variant="primary"
        aside={<Activity className="size-4 text-success" />}
    />
)
