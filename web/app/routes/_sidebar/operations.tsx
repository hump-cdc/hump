import { createFileRoute } from '@tanstack/react-router'

import { Operations } from '@/pages/operations'

export const Route = createFileRoute('/_sidebar/operations')({
    component: Operations,
    staticData: { title: 'Operations' },
})
