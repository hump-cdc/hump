import { createFileRoute } from '@tanstack/react-router'

import { Configuration } from '@/pages/configuration'

export const Route = createFileRoute('/_sidebar/configuration')({
    component: Configuration,
    staticData: {
        title: 'Configuration',
    },
})
