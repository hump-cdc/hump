import type { FC } from 'react'

import { Plug } from 'lucide-react'

import { MetricCard } from '@/shared/ui/card'

const VALUES = [8, 9, 8, 10, 9, 8, 9, 10].map((value, index) => ({ date: `16:0${index}`, value }))

export const OpenFds: FC = () => (
    <MetricCard
        title="Open FDs"
        value="96"
        values={VALUES}
        variant="primary"
        aside={<Plug className="size-4 text-ink-subtle" />}
    />
)
