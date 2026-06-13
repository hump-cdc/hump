import type { FC } from 'react'

import { GitBranch } from 'lucide-react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [16, 18, 17, 20, 19, 18, 21, 20].map((value, index) => ({ date: `16:0${index}`, value }))

export const Goroutines: FC = () => (
    <MetricCard
        title="Goroutines"
        value="248"
        values={VALUES}
        variant="primary"
        aside={<GitBranch className="size-4 text-primary" />}
    />
)
