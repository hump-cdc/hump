import { createFileRoute } from '@tanstack/react-router'

import { SnapshotPage } from '@/pages/snapshot'

export const Route = createFileRoute('/_sidebar/snapshot')({
    component: SnapshotPage,
    staticData: { title: 'Snapshot' },
})
