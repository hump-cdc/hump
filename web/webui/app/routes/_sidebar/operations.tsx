import { createFileRoute } from '@tanstack/react-router'

import { Operations, operationsSearchSchema } from '@/pages/operations'

export const Route = createFileRoute('/_sidebar/operations')({
    component: Operations,
    validateSearch: operationsSearchSchema,
    staticData: { title: 'Operations' },
})
