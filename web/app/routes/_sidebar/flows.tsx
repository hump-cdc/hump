import { createFileRoute } from '@tanstack/react-router'

import { Flows } from '@/pages/flows'

export const Route = createFileRoute('/_sidebar/flows')({
    component: Flows,
    staticData: { title: 'Flows' },
})
