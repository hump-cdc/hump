import {
    createMemoryHistory,
    createRootRoute,
    createRoute,
    createRouter,
    Outlet,
    RouterProvider,
} from '@tanstack/react-router'

import type { Meta, StoryObj } from '@storybook/react'
import { RefreshCw } from 'lucide-react'

import { Button } from '@/shared/ui/button'

import { AppLayout } from './app-layout'

const childRouteConfig = [
    { to: '/configuration', label: 'Configuration' },
    { to: '/flows', label: 'Flows' },
    { to: '/operations', label: 'Operations' },
    { to: '/stats', label: 'Stats' },
]

const rootRoute = createRootRoute({
    component: () => (
        <AppLayout>
            <Outlet />
        </AppLayout>
    ),
})

const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    staticData: {
        title: 'Overview',
        headerAside: (
            <Button variant="outline">
                <RefreshCw />
                Refresh
            </Button>
        ),
    },
    component: () => (
        <div className="flex flex-1 flex-col gap-4 p-6">
            <div className="h-24 rounded-xl border border-hairline bg-surface-1" />
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div className="h-32 rounded-xl border border-hairline bg-surface-1" />
                <div className="h-32 rounded-xl border border-hairline bg-surface-1" />
                <div className="h-32 rounded-xl border border-hairline bg-surface-1" />
            </div>
        </div>
    ),
})

const childRoutes = childRouteConfig.map((item) =>
    createRoute({
        getParentRoute: () => rootRoute,
        path: item.to,
        staticData: { title: item.label },
        component: () => <div className="flex flex-1 flex-col gap-4 p-6 text-ink">{item.label}</div>,
    }),
)

const router = createRouter({
    routeTree: rootRoute.addChildren([indexRoute, ...childRoutes]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
})

const meta: Meta = {
    title: 'Widgets/Sidebar',
    parameters: { layout: 'fullscreen', disableSideBySide: true, disablePadding: true },
    render: () => <RouterProvider router={router} />,
}

export default meta

export const Default: StoryObj = {
    name: 'Sidebar',
}
