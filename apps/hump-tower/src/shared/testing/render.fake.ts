import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createMemoryHistory, createRootRoute, createRouter, RouterProvider } from '@tanstack/react-router'

import { createElement, type PropsWithChildren, type ReactNode } from 'react'

import { act, type RenderHookResult, type RenderResult, render, renderHook } from '@testing-library/react'

import { TooltipProvider } from '@/shared/ui/tooltip'

/** What `renderWithProviders` returns: the render result plus the query client the tree uses. */
export type ProvidersRender = RenderResult & { client: QueryClient }

/** What `renderHookWithProviders` returns: the hook result plus the query client it reads. */
export type ProvidersHookRender<T> = RenderHookResult<T, unknown> & { client: QueryClient }

/** Renders `ui` under a query client and the tooltip provider, as `Providers` wraps the app. */
export const renderWithProviders = (ui: ReactNode, client: QueryClient = createQueryClient()): ProvidersRender => ({
    ...render(wrap(ui, client)),
    client,
})

/** Renders `hook` under the same providers as `renderWithProviders`. */
export const renderHookWithProviders = <T>(
    hook: () => T,
    client: QueryClient = createQueryClient(),
): ProvidersHookRender<T> => ({
    ...renderHook(hook, { wrapper: ({ children }: PropsWithChildren) => wrap(children, client) }),
    client,
})

/**
 * Renders `ui` as the root route of a memory router, for components with a `Link`; resolves once
 * the router has rendered it.
 */
export const renderWithRouter = async (
    ui: ReactNode,
    client: QueryClient = createQueryClient(),
): Promise<ProvidersRender> => {
    const routeTree = createRootRoute({ component: () => ui })
    const router = createRouter({ routeTree, history: createMemoryHistory({ initialEntries: ['/'] }) })
    await router.load()

    let result: RenderResult | undefined
    await act(async () => {
        result = render(wrap(createElement(RouterProvider, { router }), client))
    })
    if (!result) throw new Error('The router did not render')

    return { ...result, client }
}

/** Waits for pending promises and timers of `ms` milliseconds inside `act`. */
export const flush = async (ms = 0): Promise<void> => {
    await act(async () => {
        await new Promise((resolve) => setTimeout(resolve, ms))
    })
}

/** A query client that does not retry, so a failed request settles on the first attempt. */
const createQueryClient = (): QueryClient =>
    new QueryClient({ defaultOptions: { queries: { retry: false, retryOnMount: false }, mutations: { retry: false } } })

/** Wraps `ui` in the providers every rendered slice expects. */
const wrap = (ui: ReactNode, client: QueryClient): ReactNode =>
    createElement(QueryClientProvider, { client }, createElement(TooltipProvider, null, ui))
