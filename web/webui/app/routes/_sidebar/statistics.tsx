import { createFileRoute } from '@tanstack/react-router'

import { Statistics } from '@/pages/statistics'

export const Route = createFileRoute('/_sidebar/statistics')({
    component: Statistics,
    staticData: { title: 'Statistics' },
})
