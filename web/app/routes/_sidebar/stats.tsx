import { createFileRoute } from '@tanstack/react-router'

import { Stats } from '@/pages/stats'

export const Route = createFileRoute('/_sidebar/stats')({
    component: Stats,
    staticData: { title: 'Stats' },
})
