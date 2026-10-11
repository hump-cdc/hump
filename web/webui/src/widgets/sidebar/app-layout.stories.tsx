import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
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

import { type ConfigResponseBodyModel, getApiConfigQueryKey } from '@/shared/api'
import { Button } from '@/shared/ui/button'

import { AppLayout } from './app-layout'

/** Configuration the header's connection badges render; the story makes no request. */
const config: ConfigResponseBodyModel = {
    app: { version: '0.0.0' },
    database: { host: 'localhost', port: 5432, name: 'hump', slot: 'hump_slot', publication: 'hump_pub' },
    sinks: [],
    tables: [],
}

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
queryClient.setQueryData(getApiConfigQueryKey(), config)

const childRouteConfig = [
    { to: '/configuration', label: 'Configuration' },
    { to: '/operations', label: 'Operations' },
    { to: '/snapshot', label: 'Snapshot' },
    { to: '/statistics', label: 'Statistics' },
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
        title: 'Flows',
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
    render: () => (
        <QueryClientProvider client={queryClient}>
            <RouterProvider router={router} />
        </QueryClientProvider>
    ),
}

export default meta

export const Default: StoryObj = {
    name: 'Sidebar',
}
