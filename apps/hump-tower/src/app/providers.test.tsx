import { createMemoryHistory, RouterProvider } from '@tanstack/react-router'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, test } from 'bun:test'

import { act, cleanup, fireEvent, type RenderResult, render } from '@testing-library/react'

import { defaultFilterValues, useOperationsFilter } from '@/entities/operations/filter'
import { liveStream } from '@/shared/live'
import { fakeApi, json, sse } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { configResponse, metricsResponse, operationsResponse } from '@/shared/testing/fixtures.fake'
import '@/shared/testing/live.fake'
import { flush } from '@/shared/testing/render.fake'

import { getRouter } from '../../app/router'

// The app router renders `Providers` at its root, so these tests drive the whole app: the sidebar
// layout, every page and the not-found fallback, against the fake API.

/** The per-test timeout of the `app` tests, in ms: on a slow CI runner with coverage each whole-app render takes
 * seconds, so a test that visits several pages outgrows bun's 5 s default. */
const appTestTimeout = 30_000

/** The router of the app on a memory history starting at `path`, plus its rendered tree. */
type AppRender = { router: ReturnType<typeof getRouter>; view: RenderResult }

beforeAll(() => {
    registerDom()
    // A compile-time constant in the build (src/env.d.ts); the sidebar badge reads it.
    Object.defineProperty(globalThis, 'VERSION', { value: '1.2.3', configurable: true })
})
afterAll(async () => {
    // The filter store is a module singleton; leave it as the next test file expects to find it.
    useOperationsFilter.getState().apply(defaultFilterValues)
    liveStream.stop()
    Reflect.deleteProperty(globalThis, 'VERSION')
    await unregisterDom()
})
beforeEach(() => {
    localStorage.clear()
    fakeApi.serve({
        'GET /api/config': () => json(configResponse()),
        'GET /api/snapshot': (request) => sse(request, [], true),
        'POST /api/metrics': () => json(metricsResponse()),
        'POST /api/operations': () => json(operationsResponse([])),
    })
})
afterEach(cleanup)

describe('app', () => {
    test(
        'renders the flows page in the sidebar layout once the configuration loads',
        async () => {
            const { view } = await renderApp('/')

            expect(view.getByText('1.2.3')).toBeDefined()
            expect(view.getAllByText('localhost:5432').length).toBeGreaterThan(0)
            expect(view.getAllByText('Flows').length).toBeGreaterThan(0)
        },
        appTestTimeout,
    )

    test(
        'navigates between the pages through the sidebar menu',
        async () => {
            const { router, view } = await renderApp('/')

            for (const [label, path] of [
                ['Configuration', '/configuration'],
                ['Operations', '/operations'],
                ['Snapshot', '/snapshot'],
                ['Statistics', '/statistics'],
                ['Flows', '/'],
            ] as const) {
                await click(menuLink(view, label))
                expect(router.state.location.pathname).toBe(path)
            }
        },
        appTestTimeout,
    )

    test(
        'returns home through the logo link',
        async () => {
            const { router, view } = await renderApp('/statistics')

            await click(view.getByAltText('hump'))

            expect(router.state.location.pathname).toBe('/')
        },
        appTestTimeout,
    )

    test(
        'presets the operations filter from the search params and drops them from the URL',
        async () => {
            const { router } = await renderApp('/operations?table=users')

            expect(router.state.location.pathname).toBe('/operations')
            expect(router.state.location.searchStr).toBe('')
        },
        appTestTimeout,
    )

    test(
        'toggles the theme from the sidebar footer',
        async () => {
            const { view } = await renderApp('/')

            await click(view.getByText('Dark Theme'))

            expect(view.getByText('Light Theme')).toBeDefined()
            expect(localStorage.getItem('theme')).toBe('light')
        },
        appTestTimeout,
    )

    test(
        'shows the expand trigger in the header once the sidebar collapses',
        async () => {
            const { view } = await renderApp('/')
            const triggers = view.getAllByRole('button', { name: 'Toggle sidebar' }).length

            await click(view.getAllByRole('button', { name: 'Toggle sidebar' })[1])

            expect(view.getAllByRole('button', { name: 'Toggle sidebar' })).toHaveLength(triggers + 1)
        },
        appTestTimeout,
    )

    test(
        'shows the not-found page with the requested path',
        async () => {
            const { view } = await renderApp('/nope')

            expect(view.getByText('cd: no such file or directory: /nope', { exact: false })).toBeDefined()
        },
        appTestTimeout,
    )
})

/** Renders the app router on a memory history at `path` and waits for the pages to settle. */
const renderApp = async (path: string): Promise<AppRender> => {
    const router = getRouter()
    router.update({ ...router.options, history: createMemoryHistory({ initialEntries: [path] }) })
    await router.load()

    const view = render(<RouterProvider router={router} />)
    await flush(50)

    return { router, view }
}

/** The sidebar menu link labelled `label`. */
const menuLink = (view: RenderResult, label: string): HTMLElement => {
    const link = view.getAllByRole('link').find((element) => element.textContent === label)
    if (!link) throw new Error(`No menu link "${label}"`)
    return link
}

/** Clicks `element` and lets the navigation and queries it starts settle. */
const click = async (element: HTMLElement): Promise<void> => {
    await act(async () => {
        fireEvent.click(element)
    })
    await flush(50)
}
