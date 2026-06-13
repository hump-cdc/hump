import { createFileRoute, Outlet } from '@tanstack/react-router'

import type { FC } from 'react'

import { AppLayout } from '@/widgets/sidebar'

const SidebarLayout: FC = () => (
    <AppLayout>
        <Outlet />
    </AppLayout>
)

export const Route = createFileRoute('/_sidebar')({ component: SidebarLayout })
